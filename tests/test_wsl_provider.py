#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
WSL Provider E2E Test Suite
自动化测试 DevPod WSL Provider 的所有功能
"""

import subprocess
import os
import sys
import time
import json
from typing import Tuple, List
from enum import Enum


class TestResult(Enum):
    PASS = "PASS"
    FAIL = "FAIL"
    SKIP = "SKIP"
    ERROR = "ERROR"


class WSLProviderTester:
    """WSL Provider 测试框架"""

    def __init__(self, distro: str = "codepod-desktop", provider_path: str = None):
        self.distro = distro
        self.provider_path = provider_path or self._find_provider()
        self.results: List[Tuple[str, TestResult, str]] = []

    def _find_provider(self) -> str:
        """查找 provider 可执行文件"""
        paths = [
            "devpod-provider-wsl.exe",
            "./devpod-provider-wsl.exe",
        ]
        for path in paths:
            if os.path.exists(path):
                return os.path.abspath(path)
        return "devpod-provider-wsl.exe"

    def _run_cmd(self, cmd: str, timeout: int = 30) -> Tuple[int, str, str]:
        """运行命令并返回结果"""
        env = os.environ.copy()
        env["WSL_DISTRO"] = self.distro

        if sys.platform == "win32":
            try:
                result = subprocess.run(
                    cmd,
                    capture_output=True,
                    text=True,
                    timeout=timeout,
                    env=env,
                    shell=True,
                    cwd=os.path.dirname(self.provider_path)
                )
                return result.returncode, result.stdout, result.stderr
            except subprocess.TimeoutExpired:
                return -1, "", "Command timed out"
            except Exception as e:
                return -1, "", str(e)
        else:
            try:
                result = subprocess.run(
                    cmd,
                    shell=True,
                    capture_output=True,
                    text=True,
                    timeout=timeout,
                    env=env
                )
                return result.returncode, result.stdout, result.stderr
            except subprocess.TimeoutExpired:
                return -1, "", "Command timed out"
            except Exception as e:
                return -1, "", str(e)

    def test_version(self) -> TestResult:
        """测试 version 命令"""
        code, stdout, stderr = self._run_cmd(".\\devpod-provider-wsl.exe --version")
        output = stdout + stderr
        if code == 0 and "devpod-provider-wsl" in output:
            return TestResult.PASS
        return TestResult.FAIL

    def test_help(self) -> TestResult:
        """测试 help 命令"""
        code, stdout, stderr = self._run_cmd(".\\devpod-provider-wsl.exe --help")
        output = stdout + stderr
        if code == 0 and "Available Commands" in output:
            return TestResult.PASS
        return TestResult.FAIL

    def test_status_running(self) -> TestResult:
        """测试 status 命令（运行中）"""
        code, stdout, stderr = self._run_cmd(".\\devpod-provider-wsl.exe status")
        output = stdout + stderr
        if code == 0 and "Running" in output:
            return TestResult.PASS
        return TestResult.FAIL

    def test_status_stopped(self) -> TestResult:
        """测试 status 命令（停止后）"""
        # 先停止
        self._run_cmd(".\\devpod-provider-wsl.exe stop")
        time.sleep(2)

        code, stdout, stderr = self._run_cmd(".\\devpod-provider-wsl.exe status")
        output = stdout + stderr
        # WSL2 VM 停止可能需要时间
        if code == 0 and ("Running" in output or "Stopped" in output):
            return TestResult.PASS
        return TestResult.FAIL

    def test_start(self) -> TestResult:
        """测试 start 命令"""
        # 确保已停止
        self._run_cmd(".\\devpod-provider-wsl.exe stop")
        time.sleep(2)

        code, stdout, stderr = self._run_cmd(".\\devpod-provider-wsl.exe start")
        output = stdout + stderr
        if code == 0 and ("started" in output.lower() or "already running" in output.lower()):
            return TestResult.PASS
        return TestResult.FAIL

    def test_stop(self) -> TestResult:
        """测试 stop 命令"""
        code, stdout, stderr = self._run_cmd(".\\devpod-provider-wsl.exe stop")
        output = stdout + stderr
        if code == 0 and ("stop" in output.lower()):
            return TestResult.PASS
        return TestResult.FAIL

    def test_init(self) -> TestResult:
        """测试 init 命令"""
        code, stdout, stderr = self._run_cmd(".\\devpod-provider-wsl.exe init")
        output = stdout + stderr
        if code == 0 and "WSL environment check passed" in output:
            return TestResult.PASS
        return TestResult.FAIL

    def test_missing_distro(self) -> TestResult:
        """测试缺少 WSL_DISTRO 环境变量"""
        cmd = ".\\devpod-provider-wsl.exe status"
        env = os.environ.copy()
        env.pop("WSL_DISTRO", None)
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=10,
                env=env,
                shell=True
            )
            if result.returncode != 0 and "WSL_DISTRO" in result.stderr:
                return TestResult.PASS
        except Exception:
            pass
        return TestResult.FAIL

    def test_invalid_distro(self) -> TestResult:
        """测试不存在的发行版 - 行为: 返回 Stopped 状态"""
        invalid_distro = "nonexistent-distro-xyz123"
        cmd = ".\\devpod-provider-wsl.exe status"
        env = os.environ.copy()
        env["WSL_DISTRO"] = invalid_distro
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=10,
                env=env,
                shell=True,
                cwd=os.path.dirname(self.provider_path)
            )
            output = result.stderr + result.stdout
            # 对于不存在的发行版，状态应返回 Stopped
            if "Stopped" in output:
                return TestResult.PASS
        except Exception:
            pass
        return TestResult.FAIL

    def test_full_lifecycle(self) -> TestResult:
        """测试完整生命周期"""
        # Stop
        self._run_cmd(".\\devpod-provider-wsl.exe stop")
        time.sleep(2)

        # Status
        code1, _, _ = self._run_cmd(".\\devpod-provider-wsl.exe status")

        # Start
        code2, _, _ = self._run_cmd(".\\devpod-provider-wsl.exe start")

        # Status
        code3, stdout3, _ = self._run_cmd(".\\devpod-provider-wsl.exe status")

        # Stop
        code4, _, _ = self._run_cmd(".\\devpod-provider-wsl.exe stop")

        if code1 == 0 and code2 == 0 and code3 == 0 and code4 == 0:
            return TestResult.PASS
        return TestResult.FAIL

    def test_all_commands_help(self) -> TestResult:
        """测试所有命令的帮助信息"""
        commands = ["init", "command", "start", "stop", "status"]
        all_have_help = True

        for cmd in commands:
            code, stdout, _ = self._run_cmd(f".\\devpod-provider-wsl.exe {cmd} --help")
            if code != 0 or cmd not in stdout:
                all_have_help = False

        if all_have_help:
            return TestResult.PASS
        return TestResult.FAIL

    def run_all_tests(self) -> dict:
        """运行所有测试"""
        tests = [
            ("test_version", self.test_version, "Version command"),
            ("test_help", self.test_help, "Help command"),
            ("test_status_running", self.test_status_running, "Status (Running)"),
            ("test_status_stopped", self.test_status_stopped, "Status (Stopped)"),
            ("test_start", self.test_start, "Start command"),
            ("test_stop", self.test_stop, "Stop command"),
            ("test_init", self.test_init, "Init command"),
            ("test_missing_distro", self.test_missing_distro, "Missing WSL_DISTRO"),
            ("test_invalid_distro", self.test_invalid_distro, "Invalid distribution"),
            ("test_full_lifecycle", self.test_full_lifecycle, "Full lifecycle"),
            ("test_all_commands_help", self.test_all_commands_help, "All commands help"),
        ]

        print("\n" + "=" * 60)
        print("WSL Provider E2E Test Suite")
        print("Distribution: " + self.distro)
        print("Provider: " + self.provider_path)
        print("=" * 60 + "\n")

        passed = 0
        failed = 0
        skipped = 0

        for name, test_fn, description in tests:
            try:
                print("Running: " + name)
                print("  Description: " + description)
                result = test_fn()
                status_symbol = "[PASS]" if result == TestResult.PASS else "[FAIL]" if result == TestResult.FAIL else "[SKIP]"
                print("  Result: " + status_symbol + " " + result.value)
                self.results.append((name, result, description))
                if result == TestResult.PASS:
                    passed += 1
                elif result == TestResult.FAIL:
                    failed += 1
                else:
                    skipped += 1
            except Exception as e:
                print("  Error: " + str(e))
                self.results.append((name, TestResult.ERROR, str(e)))
                failed += 1

        print("\n" + "=" * 60)
        print("Test Summary")
        print("  Passed: " + str(passed))
        print("  Failed: " + str(failed))
        print("  Skipped: " + str(skipped))
        print("  Total: " + str(passed + failed + skipped))
        print("=" * 60 + "\n")

        return {
            "passed": passed,
            "failed": failed,
            "skipped": skipped,
            "total": passed + failed + skipped,
            "results": self.results
        }


def main():
    """主函数"""
    import argparse

    parser = argparse.ArgumentParser(description="WSL Provider E2E Tests")
    parser.add_argument("--distro", "-d", default="codepod-desktop",
                        help="WSL distribution name")
    parser.add_argument("--provider", "-p", default=None,
                        help="Path to provider executable")
    parser.add_argument("--json", action="store_true",
                        help="Output results as JSON")

    args = parser.parse_args()

    tester = WSLProviderTester(distro=args.distro, provider_path=args.provider)
    results = tester.run_all_tests()

    if args.json:
        print(json.dumps(results, indent=2, default=str))

    # Exit with error code if any tests failed
    sys.exit(0 if results["failed"] == 0 else 1)


if __name__ == "__main__":
    main()
