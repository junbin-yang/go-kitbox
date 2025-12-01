#!/usr/bin/env python3
import re
from collections import defaultdict

# Parse coverage.out file
pkg_stats = defaultdict(lambda: {'total': 0, 'covered': 0})

with open('coverage.out', 'r') as f:
    for line in f:
        if line.startswith('mode:'):
            continue

        parts = line.strip().split()
        if len(parts) < 3:
            continue

        file_path = parts[0].split(':')[0]
        covered = int(parts[2])

        # Extract package name
        pkg_match = re.match(r'(github\.com/junbin-yang/go-kitbox/pkg/[^/]+)', file_path)
        if pkg_match:
            pkg = pkg_match.group(1)
            statements = int(parts[1])
            pkg_stats[pkg]['total'] += statements
            if covered > 0:
                pkg_stats[pkg]['covered'] += statements

# Calculate and display coverage by package
print("Coverage by Package:")
print("-" * 80)
print(f"{'Package':<50} {'Coverage':>10} {'Status':>15}")
print("-" * 80)

results = []
for pkg, stats in sorted(pkg_stats.items()):
    if stats['total'] > 0:
        coverage = (stats['covered'] / stats['total']) * 100
        status = "Good" if coverage >= 75 else "Needs Work"
        results.append((pkg, coverage, status, stats['total'], stats['covered']))

# Sort by coverage percentage
results.sort(key=lambda x: x[1])

for pkg, coverage, status, total, covered in results:
    pkg_name = pkg.split('/')[-1]
    print(f"{pkg_name:<50} {coverage:>9.1f}% {status:>15}")

print("-" * 80)
overall_total = sum(s['total'] for s in pkg_stats.values())
overall_covered = sum(s['covered'] for s in pkg_stats.values())
overall_coverage = (overall_covered / overall_total * 100) if overall_total > 0 else 0
print(f"{'Overall':<50} {overall_coverage:>9.1f}%")
print()

# Show packages that need improvement
print("\nPackages needing improvement (< 75% coverage):")
print("-" * 80)
for pkg, coverage, status, total, covered in results:
    if coverage < 75:
        pkg_name = pkg.split('/')[-1]
        missing = total - covered
        print(f"{pkg_name:<30} {coverage:>6.1f}%  (need {missing:>4} more statements)")
