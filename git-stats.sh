#!/bin/bash

# 默认统计所有分支，也可以指定分支
branch=${1:-"--all"}

echo "--------------------------------------------------"
echo "Git 项目贡献统计 (分支: $branch)"
echo "时间范围: 全量统计"
echo "--------------------------------------------------"

# 获取所有贡献者的列表
authors=$(git log $branch --format='%aN' | sort -u)

# 打印表头
printf "%-20s %-10s %-10s %-10s %-10s\n" "作者" "提交数" "新增行" "删除行" "净增行"
echo "---------------------------------------------------------------------------"

for author in $authors; do
    # 统计提交次数
    commit_count=$(git rev-list --count --author="$author" $branch)
    
    # 统计代码行数变化
    stats=$(git log --author="$author" $branch --pretty=tformat: --numstat | \
        awk '{ add += $1; subs += $2; loc += $1 - $2 } END { printf "%s %s %s", add, subs, loc }')
    
    # 解析 awk 输出
    read -r added deleted total <<< "$stats"
    
    # 如果没有数据则补0
    added=${added:-0}
    deleted=${deleted:-0}
    total=${total:-0}

    # 打印结果
    printf "%-20s %-10s %-10s %-10s %-10s\n" "$author" "$commit_count" "$added" "$deleted" "$total"
done

echo "---------------------------------------------------------------------------"