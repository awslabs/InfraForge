// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/awslabs/InfraForge/core/partition"
	"github.com/awslabs/InfraForge/core/utils/aws"
)

func main() {
	osName := flag.String("os", "amazon", "OS name (amazon, ubuntu, debian, centos, rocky, suse, windows, redhat)")
	flag.StringVar(osName, "o", "amazon", "OS name (short)")
	osVersion := flag.String("version", "2023", "OS version (e.g., 2023, 24.04, 12, 9)")
	flag.StringVar(osVersion, "v", "2023", "OS version (short)")
	arch := flag.String("arch", "aarch64", "Architecture (x86_64, aarch64)")
	flag.StringVar(arch, "a", "aarch64", "Architecture (short)")
	region := flag.String("region", "", "AWS region (overrides default)")
	flag.StringVar(region, "r", "", "AWS region (short)")
	profile := flag.String("profile", "", "AWS profile (overrides default)")
	flag.StringVar(profile, "p", "", "AWS profile (short)")
	listAll := flag.Bool("list", false, "List all supported OS/version combinations")
	flag.BoolVar(listAll, "l", false, "List (short)")
	flag.Parse()

	if *listAll {
		printSupported()
		return
	}


	// Detect partition from region
	p := "aws"
	if *region != "" && strings.HasPrefix(*region, "cn-") {
		p = "aws-cn"
	} else if partition.DefaultPartition != "" {
		p = partition.DefaultPartition
	}

	owner, namePattern := aws.GetAMIInfo(p, *osName, *osVersion, *arch)
	if owner == "" || namePattern == "" {
		fmt.Fprintf(os.Stderr, "Unsupported combination: os=%s version=%s arch=%s partition=%s\n", *osName, *osVersion, *arch, p)
		fmt.Fprintf(os.Stderr, "Run 'ami-lookup --list' to see supported combinations.\n")
		os.Exit(1)
	}

	lookupArch := *arch
	if lookupArch == "aarch64" {
		lookupArch = "arm64"
	}

	lookup := &aws.ForgeAMILookup{
		AmiOwner: owner,
		AmiName:  namePattern,
		AmiArch:  lookupArch,
		Region:   *region,
		Profile:  *profile,
	}

	amiID, err := lookup.FindAMI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if amiID == "" {
		fmt.Fprintf(os.Stderr, "No AMI found for: os=%s version=%s arch=%s\n", *osName, *osVersion, *arch)
		os.Exit(1)
	}

	fmt.Printf("%s\n", amiID)
}

func printSupported() {
	supported := map[string][]string{
		"amazon":  {"2", "2023"},
		"ubuntu":  {"18.04", "20.04", "22.04", "24.04", "26.04"},
		"debian":  {"10", "11", "12", "13"},
		"centos":  {"7", "9", "10"},
		"rocky":   {"8", "9", "10"},
		"suse":    {"12", "15", "16"},
		"redhat":  {},
		"windows": {"2016", "2019", "2022", "2025"},
	}

	fmt.Println("Supported OS/Version combinations:")
	fmt.Println()

	names := make([]string, 0, len(supported))
	for k := range supported {
		names = append(names, k)
	}
	sort.Strings(names)

	for _, name := range names {
		versions := supported[name]
		if len(versions) > 0 {
			fmt.Printf("  %-10s %s\n", name, strings.Join(versions, ", "))
		} else {
			fmt.Printf("  %-10s (check available versions)\n", name)
		}
	}
	fmt.Println()
	fmt.Println("Architectures: x86_64, aarch64")
	fmt.Println("Note: Some older versions (CentOS 7, Debian 10) may only be available as deprecated AMIs.")
}
