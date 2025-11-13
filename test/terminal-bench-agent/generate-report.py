#!/usr/bin/env python3
"""
Generate test report from terminal-bench results
"""

import os
import json
import sys
from pathlib import Path
from collections import defaultdict


def load_results(result_dir: str):
    """Load all test results from directory"""
    results = []
    result_path = Path(result_dir)
    
    for json_file in result_path.glob("*.json"):
        with open(json_file, 'r') as f:
            try:
                data = json.load(f)
                results.append(data)
            except json.JSONDecodeError:
                print(f"Warning: Failed to parse {json_file}")
                
    return results


def analyze_results(results):
    """Analyze test results"""
    stats = {
        "total": 0,
        "passed": 0,
        "failed": 0,
        "warning": 0,
        "by_category": defaultdict(lambda: {"passed": 0, "failed": 0})
    }
    
    failed_tasks = []
    
    for result in results:
        task_name = result.get("task", "unknown")
        status = result.get("status", "failed")
        category = result.get("category", "unknown")
        
        stats["total"] += 1
        
        if status == "passed":
            stats["passed"] += 1
            stats["by_category"][category]["passed"] += 1
        elif status == "warning":
            stats["warning"] += 1
        else:
            stats["failed"] += 1
            stats["by_category"][category]["failed"] += 1
            failed_tasks.append({
                "name": task_name,
                "category": category,
                "reason": result.get("error", "Unknown error")
            })
            
    return stats, failed_tasks


def generate_markdown_report(stats, failed_tasks):
    """Generate markdown report"""
    report = []
    
    report.append("# Terminal-Bench 测试报告\n")
    report.append("## 测试摘要\n")
    
    total = stats["total"]
    passed = stats["passed"]
    failed = stats["failed"]
    warning = stats["warning"]
    pass_rate = (passed / total * 100) if total > 0 else 0
    
    report.append(f"- **总任务数**: {total}")
    report.append(f"- **通过数**: {passed} ✅")
    report.append(f"- **失败数**: {failed} ❌")
    report.append(f"- **警告数**: {warning} ⚠️")
    report.append(f"- **通过率**: {pass_rate:.2f}%\n")
    
    report.append("## 分类统计\n")
    report.append("| 类别 | 通过 | 失败 | 通过率 |")
    report.append("|------|------|------|--------|")
    
    for category, cat_stats in stats["by_category"].items():
        cat_passed = cat_stats["passed"]
        cat_failed = cat_stats["failed"]
        cat_total = cat_passed + cat_failed
        cat_rate = (cat_passed / cat_total * 100) if cat_total > 0 else 0
        report.append(f"| {category} | {cat_passed} | {cat_failed} | {cat_rate:.1f}% |")
    
    if failed_tasks:
        report.append("\n## 失败任务详情\n")
        report.append("| 任务名称 | 类别 | 失败原因 |")
        report.append("|---------|------|---------|")
        
        for task in failed_tasks:
            name = task["name"]
            category = task["category"]
            reason = task["reason"][:100] + "..." if len(task["reason"]) > 100 else task["reason"]
            report.append(f"| {name} | {category} | {reason} |")
    
    report.append("\n## 改进建议\n")
    
    if pass_rate < 70:
        report.append("- ⚠️ 通过率较低，建议使用 `conservative` 模式")
        report.append("- 提高 `compressionTokenThreshold` 到 800")
        report.append("- 启用 `preserveKeyInfo: true`")
    elif pass_rate < 85:
        report.append("- 通过率中等，建议针对失败任务类型优化配置")
        report.append("- 考虑对特定类别使用不同的压缩策略")
    else:
        report.append("- ✅ 通过率良好，当前配置适合 Agent 场景")
    
    return "\n".join(report)


def main():
    if len(sys.argv) < 2:
        print("Usage: python generate-report.py <result_directory>")
        sys.exit(1)
        
    result_dir = sys.argv[1]
    
    if not os.path.exists(result_dir):
        print(f"Error: Directory {result_dir} does not exist")
        sys.exit(1)
        
    print("Loading test results...")
    results = load_results(result_dir)
    
    if not results:
        print("No results found")
        sys.exit(1)
        
    print(f"Loaded {len(results)} test results")
    
    print("Analyzing results...")
    stats, failed_tasks = analyze_results(results)
    
    print("Generating report...")
    report = generate_markdown_report(stats, failed_tasks)
    
    # Save report
    report_file = os.path.join(result_dir, "report.md")
    with open(report_file, 'w') as f:
        f.write(report)
        
    print(f"\nReport saved to: {report_file}")
    print("\n" + "="*50)
    print(report)
    print("="*50)


if __name__ == "__main__":
    main()

