// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package aws

import "testing"

// TestAmiNameVersion 固化「从镜像名提取数字序列」的行为。
// 这些样本全部取自 2026-08 实际的 DescribeImages 输出。
func TestAmiNameVersion(t *testing.T) {
	cases := []struct {
		name string
		want []int
	}{
		// RHEL：三段式与两段式并存，是唯一会跨小版本的模式。
		// 数字逐位取出，前两位就足以分出小版本高低。
		{"RHEL-9.8.0_HVM-20260728-x86_64-0-Hourly2-GP3", []int{9, 8, 0, 20260728, 86, 64, 0, 2, 3}},
		{"RHEL-8.8_HVM-20241008-arm64-2259-Hourly2-GP3", []int{8, 8, 20241008, 64, 2259, 2, 3}},
		{"RHEL-7.9_HVM-20240930-x86_64-0-Hourly2-GP3", []int{7, 9, 20240930, 86, 64, 0, 2, 3}},
		// 其他发行版：名字里已写死版本，候选之间只有构建日期不同，
		// 逐位比较最终落在日期那一位上，等价于改动前「按发布时间取最新」。
		// 注意 gp3 的 3 会出现在序列里 —— 但同族候选都有它，不影响相对顺序。
		{"ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-20260805", []int{3, 24, 4, 64, 20260805}},
		{"CentOS Stream 9 x86_64 20260401", []int{9, 86, 64, 20260401}},
		{"suse-sles-15-sp5-v20260101-hvm-ssd-arm64", []int{15, 5, 20260101, 64}},
		{"no-digits-here", nil},
	}
	for _, c := range cases {
		got := amiNameVersion(c.name)
		if len(got) != len(c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s: got %v, want %v", c.name, got, c.want)
				break
			}
		}
	}
}

// TestAmiNameVersionOrdering 直接固化「选谁」的语义：
// 同一大版本下，9.8 必须排在 9.6 之前，哪怕 9.6 的构建日期更晚（EUS 回头重发）。
func TestAmiNameVersionOrdering(t *testing.T) {
	newer := "RHEL-9.8.0_HVM-20260728-x86_64-0-Hourly2-GP3"  // 发布于 2026-07-29
	older := "RHEL-9.6.0_HVM-20260811-x86_64-0-Hourly2-GP3"  // 发布于 2026-08-11（更晚）
	if cmpVersion(amiNameVersion(newer), amiNameVersion(older)) != 1 {
		t.Errorf("9.8 应当胜过 9.6（即使 9.6 发布时间更晚）")
	}
	// 8.10 > 8.8：按数值比较，字符串比较会得到相反结果
	if cmpVersion(amiNameVersion("RHEL-8.10.0_HVM-20260721-x86_64-2225-Hourly2-GP3"),
		amiNameVersion("RHEL-8.8_HVM-20241008-x86_64-2247-Hourly2-GP3")) != 1 {
		t.Errorf("8.10 应当胜过 8.8")
	}
	// 10.2 > 10.0
	if cmpVersion(amiNameVersion("RHEL-10.2.0_HVM-20260728-arm64-0-Hourly2-GP3"),
		amiNameVersion("RHEL-10.0.0_HVM-20260812-arm64-0-Hourly2-GP3")) != 1 {
		t.Errorf("10.2 应当胜过 10.0")
	}
	// 同一小版本内，更新的构建（日期更大）胜出
	if cmpVersion(amiNameVersion("RHEL-9.8.0_HVM-20260728-x86_64-0-Hourly2-GP3"),
		amiNameVersion("RHEL-9.8.0_HVM-20260101-x86_64-0-Hourly2-GP3")) != 1 {
		t.Errorf("同一小版本内应当取更新的构建")
	}
}

// TestAmiNameVersionAl2023Kernel 固化一个副作用：
// AL2023 同一天会同时发布 kernel 6.1 / 6.12 / 6.18 三个分支，CreationDate 完全相同。
// 改动前用的是「严格晚于」比较，时间相同时保留先遇到的那张 —— 也就是取决于
// DescribeImages 的返回顺序（AWS 不保证），选到哪个内核版本其实是运气。
// 现在按名字里的数字比较，确定性地取最高内核版本（6.18 > 6.12 > 6.1）。
func TestAmiNameVersionAl2023Kernel(t *testing.T) {
	k618 := amiNameVersion("al2023-ami-2023.12.20260817.0-kernel-6.18-arm64")
	k612 := amiNameVersion("al2023-ami-2023.12.20260817.0-kernel-6.12-arm64")
	k61 := amiNameVersion("al2023-ami-2023.12.20260817.0-kernel-6.1-arm64")
	if cmpVersion(k618, k612) != 1 || cmpVersion(k612, k61) != 1 {
		t.Errorf("同日发布的 AL2023 应当按内核版本 6.18 > 6.12 > 6.1 取最高")
	}
	// 更新的构建日期优先于更高的内核版本（日期在数字序列里更靠前）
	newBuild := amiNameVersion("al2023-ami-2023.12.20260817.0-kernel-6.1-arm64")
	oldBuild := amiNameVersion("al2023-ami-2023.12.20260803.3-kernel-6.18-arm64")
	if cmpVersion(newBuild, oldBuild) != 1 {
		t.Errorf("更新的构建日期应当优先于更高的内核版本")
	}
}

// TestCmpVersion 覆盖「8.10 > 8.8」这类按数值而非字符串比较的关键场景。
func TestCmpVersion(t *testing.T) {
	cases := []struct {
		a, b []int
		want int
	}{
		{[]int{9, 8, 0}, []int{9, 6, 0}, 1},   // 9.8 > 9.6：改动的核心诉求
		{[]int{8, 10, 0}, []int{8, 8}, 1},     // 8.10 > 8.8（字符串比较会反过来）
		{[]int{10, 2, 0}, []int{10, 0, 0}, 1}, // 10.2 > 10.0
		{[]int{9, 6, 0}, []int{9, 6, 0}, 0},   // 相等 -> 交给发布时间决定
		{[]int{9, 6}, []int{9, 6, 0}, 0},      // 缺位补 0
		{[]int{9, 6}, []int{9, 6, 1}, -1},
		{nil, []int{1}, -1},
		{nil, nil, 0},
	}
	for _, c := range cases {
		if got := cmpVersion(c.a, c.b); got != c.want {
			t.Errorf("cmpVersion(%v, %v) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// TestGetAMIInfoRedhat 确认两个 partition 都能解析 redhat，且 owner 与镜像名模式配套。
// RHEL 7 只有 x86_64；aarch64 应当明确解析失败而不是回落到 x86_64 的镜像。
func TestGetAMIInfoRedhat(t *testing.T) {
	wantOwner := map[string]string{"aws": "309956199498", "aws-cn": "841258680906"}
	for _, part := range []string{"aws", "aws-cn"} {
		for _, ver := range []string{"8", "9", "10"} {
			for _, arch := range []string{"x86_64", "aarch64"} {
				owner, name := GetAMIInfo(part, "redhat", ver, arch)
				if owner != wantOwner[part] {
					t.Errorf("%s/redhat/%s/%s: owner = %q, want %q", part, ver, arch, owner, wantOwner[part])
				}
				if name == "" {
					t.Errorf("%s/redhat/%s/%s: name pattern is empty", part, ver, arch)
				}
			}
		}
		// RHEL 7：仅 x86_64
		if _, name := GetAMIInfo(part, "redhat", "7", "x86_64"); name == "" {
			t.Errorf("%s/redhat/7/x86_64: name pattern is empty", part)
		}
		if _, name := GetAMIInfo(part, "redhat", "7", "aarch64"); name != "" {
			t.Errorf("%s/redhat/7/aarch64: expected no pattern (Red Hat 未发布该组合), got %q", part, name)
		}
	}
}
