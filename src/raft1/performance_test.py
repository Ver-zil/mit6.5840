#!/usr/bin/env python3

import itertools
import math
import signal
import subprocess
import tempfile
import shutil
import time
import os
import sys
import datetime
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor, wait, FIRST_COMPLETED
from dataclasses import dataclass
from pathlib import Path
from typing import List, Optional, Dict, DefaultDict, Tuple
import re

import typer
import rich
from rich import print
from rich.table import Table
from rich.progress import (
    Progress,
    TimeElapsedColumn,
    TimeRemainingColumn,
    TextColumn,
    BarColumn,
    SpinnerColumn,
)
from rich.live import Live
from rich.panel import Panel
from rich.traceback import install

install(show_locals=True)
os.chdir(os.path.dirname(os.path.abspath(__file__)))  # 切换到脚本所在目录


# 统计面版，增量式计算均值、方差(welford)
@dataclass
class StatsMeter:
    """
    Auxiliary classs to keep track of online stats including: count, mean, variance
    Uses Welford's algorithm to compute sample mean and sample variance incrementally.
    https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance#On-line_algorithm
    """

    n: int = 0
    mean: float = 0.0
    S: float = 0.0

    def add(self, datum):
        self.n += 1
        delta = datum - self.mean
        # Mk = Mk-1+ (xk – Mk-1)/k
        self.mean += delta / self.n
        # Sk = Sk-1 + (xk – Mk-1)*(xk – Mk).
        self.S += delta * (datum - self.mean)

    @property
    def variance(self):
        return self.S / self.n

    @property
    def std(self):
        return math.sqrt(self.variance)


def print_results(results: Dict[str, Dict[str, StatsMeter]], official_comparsion: Dict[str, float], timing=False):
    table = Table(show_header=True, header_style="bold")
    table.add_column("Test")
    table.add_column("Failed", justify="right")
    table.add_column("Total", justify="right")
    if not timing:
        table.add_column("Time", justify="right")
    else:
        table.add_column("Real Time", justify="right")
        table.add_column("User Time", justify="right")
        table.add_column("System Time", justify="right")
    if len(official_comparsion) > 1:
        table.add_column("Offical Comparsion", justify="right")

    for test, stats in results.items():
        if stats["completed"].n == 0:
            continue
        color = "green" if stats["failed"].n == 0 else "red"
        row = [
            f"[{color}]{test}[/{color}]",
            str(stats["failed"].n),
            str(stats["completed"].n),
        ]
        if not timing:
            row.append(f"{stats['time'].mean:.2f} ± {stats['time'].std:.2f}")
        else:
            row.extend(
                [
                    f"{stats['real_time'].mean:.2f} ± {stats['real_time'].std:.2f}",
                    f"{stats['user_time'].mean:.2f} ± {stats['user_time'].std:.2f}",
                    f"{stats['system_time'].mean:.2f} ± {stats['system_time'].std:.2f}",
                ]
            )
        if len(official_comparsion) > 1:
            row.append(f"{official_comparsion[test]}")
        table.add_row(*row)

    print(table)


# 测试lab的核心部分
def run_test(test: str, race: bool, timing: bool, debug: bool):
    test_cmd = ["go", "test", f"-run={test}"]
    if race:
        test_cmd.append("-race")
    if timing:
        test_cmd = ["time"] + test_cmd
    if debug:
        test_cmd = ["DEBUG=1"] + test_cmd
    f, path = tempfile.mkstemp()
    start = time.time()
    proc = subprocess.run(test_cmd, stdout=f, stderr=f)
    runtime = time.time() - start
    os.close(f)
    return test, path, proc.returncode, runtime


def last_line(file: str) -> str:
    with open(file, "rb") as f:
        f.seek(-2, os.SEEK_END)
        while f.read(1) != b"\n":
            f.seek(-2, os.SEEK_CUR)
        line = f.readline().decode()
    return line


def get_raft_3A_test() -> Dict[str, float]:
    return {
        "TestInitialElection3A": 3.6,
        "TestReElection3A": 7.6,
        "TestManyElections3A": 8.4,
    }


def get_raft_3B_test() -> Dict[str, float]:
    return {
        "TestBasicAgree3B": 1.3,
        "TestRPCBytes3B": 2.8,
        "TestFollowerFailure3B": 5.3,
        "TestLeaderFailure3B": 6.4,
        "TestFailAgree3B": 5.9,
        "TestFailNoAgree3B": 4.3,
        "TestConcurrentStarts3B": 1.5,
        "TestRejoin3B": 5.3,
        "TestBackup3B": 12.1,
        "TestCount3B": 3.1,
    }


def get_raft_3C_test() -> Dict[str, int]:
    return {
        "TestPersist13C": 6.6,
        "TestPersist23C": 15.6,
        "TestPersist33C": 3.1,
        "TestFigure83C": 33.7,
        "TestUnreliableAgree3C": 2.1,
        "TestFigure8Unreliable3C": 31.9,
        "TestReliableChurn3C": 16.8,
        "TestUnreliableChurn3C": 16.1,
    }


def get_raft_3D_test() -> Dict[str, int]:
    return {
        "TestSnapshotBasic3D": 3.3,
        "TestSnapshotInstall3D": 48.4,
        "TestSnapshotInstallUnreliable3D": 56.1,
        "TestSnapshotInstallCrash3D": 33.3,
        "TestSnapshotInstallUnCrash3D": 38.1,
        "TestSnapshotAllCrash3D": 11.2,
        "TestSnapshotInit3D": 4.3,
    }


def get_raft_all_test() -> Dict[str, int]:
    return {
        **get_raft_3A_test(),
        **get_raft_3B_test(),
        **get_raft_3C_test(),
        **get_raft_3D_test(),
    }


def get_official_test(tests: List[str], official_tests: List[str]) -> List[str]:
    """
    根据Go风格的测试匹配规则筛选测试
    :return: 去重且保序的官方测试名列表
    """
    matched = set()
    result = []
    
    for test in official_tests:
        for pattern in tests:
            if re.search(fr"{pattern}", test) and test not in matched:
                matched.add(test)
                result.append(test)
                break  # 匹配到一个模式即可
    
    return result


# run tests的多线程本质是开多个监控面版，底层的每个go配置仍然是跑4个处理器
def run_tests(
    tests:List[str]        = typer.Argument(None, help="Test pattern (e.g. '3A 3B 3C 3D')"),
    sequential: bool       = typer.Option(False,  '--sequential',      '-s',    help='Run all test of each group in order'),
    workers: int           = typer.Option(1,      '--workers',         '-p',    help='Number of parallel tasks'),
    iterations: int        = typer.Option(10,     '--iter',            '-n',    help='Number of iterations to run'),
    output: Optional[Path] = typer.Option(None,   '--output',          '-o',    help='Output path to use'),
    verbose: int           = typer.Option(0,      '--verbose',         '-v',    help='Verbosity level', count=True),
    archive: bool          = typer.Option(False,  '--archive',         '-a',    help='Save all logs intead of only failed ones'),
    race: bool             = typer.Option(False,  '--race/--no-race',  '-r/-R', help='Run with race checker'),
    loop: bool             = typer.Option(False,  '--loop',            '-l',    help='Run continuously'),
    growth: int            = typer.Option(10,     '--growth',          '-g',    help='Growth ratio of iterations when using --loop'),
    timing: bool           = typer.Option(False,  '--timing',          '-t',    help='Report timing, only works on macOS'),
    debug: bool            = typer.Option(True,   '--debug',           '-d',    help='Disable debug mode(need to add env var in util.go)'),
    match: bool            = typer.Option(True,   '--match',           '-m',    help='Disable auto match'),
    # raft: bool             = typer.Option(False,  '--raft',            '-f',    help='test all raft tests separately'),
    # fmt: on
):

    official_data = get_raft_all_test()
    official_comparsion = {}
    if tests is None:
        tests = official_data.keys()
    
    if match:
        tests = get_official_test(tests, official_data.keys())
        official_comparsion = {test: official_data.get(test, 0) for test in tests}

    if output is None:
        timestamp = datetime.datetime.now().strftime("%Y%m%d_%H%M%S")
        output = Path("err_log/" + timestamp)

    if race:
        print("[yellow]Running with the race detector\n[/yellow]")

    if verbose > 0:
        print(f"[yellow] Verbosity level set to {verbose}[/yellow]")
        os.environ['VERBOSE'] = str(verbose)

    while True:

        total = iterations * len(tests)
        completed = 0

        results = {test: defaultdict(StatsMeter) for test in tests}

        if sequential:
            test_instances = itertools.chain.from_iterable(itertools.repeat(test, iterations) for test in tests)
        else:
            test_instances = itertools.chain.from_iterable(itertools.repeat(tests, iterations))
        test_instances = iter(test_instances)

        # 定义动态进度条
        total_progress = Progress(
            "[progress.description]{task.description}",
            BarColumn(),
            TimeRemainingColumn(),
            "[progress.percentage]{task.percentage:>3.0f}%",
            TimeElapsedColumn(),
        )
        total_task = total_progress.add_task("[yellow]Tests[/yellow]", total=total)

        task_progress = Progress(
            "[progress.description]{task.description}",
            SpinnerColumn(),
            BarColumn(),
            "[red]{task.fields[failed]}[/red]/{task.completed}/{task.total}",
            expand=True,  # 关键！允许动态字段，增加动态失败数显示
        )
        tasks = {test: task_progress.add_task(test, total=iterations, failed=0) for test in tests}

        progress_table = Table.grid()
        progress_table.add_row(total_progress)
        progress_table.add_row(Panel.fit(task_progress))

        with Live(progress_table, transient=True) as live:

            def handler(_, frame):
                live.stop()
                print('\n')
                print_results(results, official_comparsion)
                sys.exit(1)

            signal.signal(signal.SIGINT, handler)

            with ThreadPoolExecutor(max_workers=workers) as executor:

                futures = []
                while completed < total:
                    n = len(futures)
                    if n < workers:
                        for test in itertools.islice(test_instances, workers-n):
                            futures.append(executor.submit(run_test, test, race, timing, debug))

                    done, not_done = wait(futures, return_when=FIRST_COMPLETED)

                    for future in done:
                        test, path, rc, runtime = future.result()

                        results[test]['completed'].add(1)
                        results[test]['time'].add(runtime)
                        task_progress.update(tasks[test], advance=1)
                        dest = (output / f"{test}_{completed}.log").as_posix()
                        # 根据结果重新刷新进度条颜色
                        if rc != 0:
                            print(f"Failed test {test} - {dest}")
                            results[test]['failed'].add(1)
                            task_progress.update(tasks[test], description=f"[red]{test}[/red]", failed=results[test]['failed'].n)
                        else:
                            if results[test]['completed'].n == iterations and results[test]['failed'].n == 0:
                                task_progress.update(tasks[test], description=f"[green]{test}[/green]")

                        # 日志存放
                        if rc != 0 or archive:
                            output.mkdir(exist_ok=True, parents=True)
                            shutil.copy(path, dest)

                        if timing:
                            line = last_line(path)
                            real, _, user, _, system, _ = line.replace(' '*8, '').split(' ')
                            results[test]['real_time'].add(float(real))
                            results[test]['user_time'].add(float(user))
                            results[test]['system_time'].add(float(system))

                        # 手动清理临时文件
                        os.remove(path)

                        completed += 1
                        total_progress.update(total_task, advance=1)

                    futures = list(not_done)

        # 增加汇总条目，不在运行时展示
        results['Sum'] = defaultdict(StatsMeter)
        results['Sum']['completed'].n = sum([results[test]['completed'].n for test in tests])
        results['Sum']['failed'].n = sum([results[test]['failed'].n for test in tests])
        results['Sum']['time'].n = iterations
        results['Sum']['time'].mean = sum([results[test]['time'].mean for test in tests])
        results['Sum']['time'].S = sum([results[test]['time'].S for test in tests])
        if timing:
            results['Sum']['real_time'].n = iterations
            results['Sum']['user_time'].n = iterations
            results['Sum']['system_time'].n = iterations
            results['Sum']['real_time'].mean = sum([results[test]['real_time'].mean for test in tests])
            results['Sum']['user_time'].mean = sum([results[test]['user_time'].mean for test in tests])
            results['Sum']['system_time'].mean = sum([results[test]['system_time'].mean for test in tests])
            results['Sum']['real_time'].S = sum([results[test]['real_time'].S for test in tests])
            results['Sum']['user_time'].S = sum([results[test]['user_time'].S for test in tests])
            results['Sum']['system_time'].S = sum([results[test]['system_time'].S for test in tests])
        official_comparsion['Sum'] = sum([official_comparsion.get(test, 0) for test in tests])
        print_results(results, official_comparsion, timing)

        if loop:
            iterations *= growth
            print(f"[yellow]Increasing iterations to {iterations}[/yellow]")
        else:
            break


if __name__ == "__main__":
    typer.run(run_tests)
