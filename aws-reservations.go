package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/artyom/autoflags"
	"github.com/stripe/aws-go/aws"
	"github.com/stripe/aws-go/gen/ec2"
	"github.com/stripe/aws-go/gen/rds"
)

var (
	ec2fmt = "%20s\t%5s\t%d\n"
	rdsfmt = "%20s\t%10s\t%9s\t%d\n"
)

func main() {
	log.SetFlags(0)
	config := struct {
		AccessKey string `flag:"accesskey,access key (or use AWS_ACCESS_KEY_ID/AWS_ACCESS_KEY env.vars)"`
		SecretKey string `flag:"secretkey,secret key (or use AWS_SECRET_ACCESS_KEY/AWS_SECRET_KEY env.vars)"`
		Region    string `flag:"region,aws region"`
	}{
		Region: "us-west-1",
	}
	if err := autoflags.Define(&config); err != nil {
		log.Fatal(err)
	}
	flag.Parse()
	creds := aws.DetectCreds(config.AccessKey, config.SecretKey, "")

	ei := make(map[ec2Inst]int)
	ri := make(map[rdsInst]int)

	// at first fill ei and ri with running instances info, then subtract
	// reserved instances info from this data
	runningEi, err := getRunningEC2Instances(creds, config.Region)
	if err != nil {
		log.Fatal(err)
	}
	for _, ii := range runningEi {
		if ii.State != Active {
			continue
		}
		ei[ii.ec2Inst] += ii.Count
	}
	runningRi, err := getRunningRDSInstances(creds, config.Region)
	if err != nil {
		log.Fatal(err)
	}
	for _, ii := range runningRi {
		if ii.State != Active {
			continue
		}
		ri[ii.rdsInst] += ii.Count
	}

	reservedEi, err := getReservedEC2Instances(creds, config.Region)
	if err != nil {
		log.Fatal(err)
	}
	for _, ii := range reservedEi {
		if ii.State != Active {
			continue
		}
		ei[ii.ec2Inst] -= ii.Count
	}
	reservedRi, err := getReservedRDSInstances(creds, config.Region)
	if err != nil {
		log.Fatal(err)
	}
	for _, ii := range reservedRi {
		if ii.State != Active {
			continue
		}
		ri[ii.rdsInst] -= ii.Count
	}

	headerPrinted := false
	// only print active instances without matching reservations
	for k, v := range ei {
		if v < 1 {
			continue
		}
		if !headerPrinted {
			headerPrinted = true
			fmt.Println("\nOn-demand EC2 instances:")
		}
		fmt.Printf(ec2fmt, k.Class, stringVPC(k.VPC), v)
	}
	// only print reserved instances without matching running instances
	headerPrinted = false
	for k, v := range ei {
		if v >= 0 {
			continue
		}
		if !headerPrinted {
			headerPrinted = true
			fmt.Println("\nUnused EC2 reservations:")
		}
		fmt.Printf(ec2fmt, k.Class, stringVPC(k.VPC), -v)
	}

	// only print active RDS instances without matching reservations
	headerPrinted = false
	for k, v := range ri {
		if v < 1 {
			continue
		}
		if !headerPrinted {
			headerPrinted = true
			fmt.Println("\nOn-demand RDS instances:")
		}
		fmt.Printf(rdsfmt, k.Class, k.Product, stringMultiAZ(k.MultiAZ), v)
	}
	// only print reserved RDS instances without matching active instances
	headerPrinted = false
	for k, v := range ri {
		if v >= 0 {
			continue
		}
		if !headerPrinted {
			headerPrinted = true
			fmt.Println("\nUnused RDS reservation:")
		}
		fmt.Printf(rdsfmt, k.Class, k.Product, stringMultiAZ(k.MultiAZ), -v)
	}
}

func getRunningEC2Instances(creds aws.CredentialsProvider, region string) ([]ec2InstInfo, error) {
	resp, err := ec2.New(creds, region, nil).DescribeInstances(nil)
	if err != nil {
		return nil, err
	}
	var out []ec2InstInfo
	for _, r := range resp.Reservations {
		for _, inst := range r.Instances {
			out = append(out, ec2iToec2ii(inst))
		}
	}
	return out, nil
}

func getRunningRDSInstances(creds aws.CredentialsProvider, region string) ([]rdsInstInfo, error) {
	resp, err := rds.New(creds, region, nil).DescribeDBInstances(nil)
	if err != nil {
		return nil, err
	}
	var out []rdsInstInfo
	for _, r := range resp.DBInstances {
		out = append(out, rdsiTordsii(r))
	}
	return out, nil
}

func getReservedRDSInstances(creds aws.CredentialsProvider, region string) ([]rdsInstInfo, error) {
	resp, err := rds.New(creds, region, nil).DescribeReservedDBInstances(nil)
	if err != nil {
		return nil, err
	}
	var out []rdsInstInfo
	for _, r := range resp.ReservedDBInstances {
		out = append(out, rdsriTordsii(r))
	}
	return out, nil
}

func getReservedEC2Instances(creds aws.CredentialsProvider, region string) ([]ec2InstInfo, error) {
	resp, err := ec2.New(creds, region, nil).DescribeReservedInstances(nil)
	if err != nil {
		return nil, err
	}
	var out []ec2InstInfo
	for _, r := range resp.ReservedInstances {
		out = append(out, ec2riToec2ii(r))
	}
	return out, nil
}

// rdsiTordsii converts rds.DBInstance to rdsInstInfo; count always set to 1 and
// state set to Active.
func rdsiTordsii(r rds.DBInstance) rdsInstInfo {
	out := rdsInstInfo{
		rdsInst: rdsInst{
			Class:   toStr(r.DBInstanceClass),
			Product: toStr(r.Engine),
			MultiAZ: toBool(r.MultiAZ),
		},
		Count: 1,
		State: Active,
	}
	return out
}

// rdsriTordsii converts rds.ReservedDBInstance to rdsInstInfo
func rdsriTordsii(r rds.ReservedDBInstance) rdsInstInfo {
	out := rdsInstInfo{
		rdsInst: rdsInst{
			Class:   toStr(r.DBInstanceClass),
			Product: toStr(r.ProductDescription),
			MultiAZ: toBool(r.MultiAZ),
		},
		Count: toInt(r.DBInstanceCount),
	}
	switch toStr(r.State) {
	case "active":
		out.State = Active
	}
	return out
}

// ec2iToec2ii converts ec2.Instance to ec2InstInfo. Count is always set to 1,
// State set to Active for all known states except for the terminated.
func ec2iToec2ii(r ec2.Instance) ec2InstInfo {
	out := ec2InstInfo{
		ec2Inst: ec2Inst{
			Class: toStr(r.InstanceType),
			VPC:   len(toStr(r.VPCID)) > 0,
		},
		Count: 1,
		State: UnknownState,
	}
	if r.State != nil {
		switch toStr(r.State.Name) {
		case ec2.InstanceStateNameRunning,
			ec2.InstanceStateNameStopped,
			ec2.InstanceStateNameStopping,
			ec2.InstanceStateNameShuttingDown,
			ec2.InstanceStateNamePending:
			out.State = Active
		}
	}
	return out
}

// ec2riToec2ii converts ec2.ReservedInstances to ec2InstInfo
func ec2riToec2ii(r ec2.ReservedInstances) ec2InstInfo {
	out := ec2InstInfo{
		ec2Inst: ec2Inst{
			Class: toStr(r.InstanceType),
		},
		Count: toInt(r.InstanceCount),
	}
	out.VPC = strings.Contains(toStr(r.ProductDescription), "Amazon VPC")
	switch toStr(r.State) {
	case "active":
		out.State = Active
	}
	return out
}

// ec2InstInfo describes a group of ec2 instances having the same state
type ec2InstInfo struct {
	ec2Inst
	Count int   // number of instances in group
	State state // state of instances in group
}

// ec2Inst describes single ec2 instance
type ec2Inst struct {
	Class string // instance class (i.e. m3.large)
	VPC   bool   // instance belongs to VPC
}

// rdsInstInfo describes a group of RDS instances having the same state
type rdsInstInfo struct {
	rdsInst
	Count int   // number of instances in group
	State state // state of instances in group
}

// rdsInst describes single RDS instance
type rdsInst struct {
	Class   string // instance class (i.e. db.m3.large)
	Product string // type of database (mysql, postgres)
	MultiAZ bool   // instance spans multiple availability zones
}

type state uint8

const (
	UnknownState = iota
	Active
)

func (s state) String() string {
	switch s {
	case Active:
		return "active"
	}
	return "unsupported state"
}

// toInt unpacks aws.IntegerValue to integer; nil value corresponds to 0.
func toInt(i aws.IntegerValue) int {
	if i == nil {
		return 0
	}
	return *i
}

// toStr unpacks aws.StringValue to string; nil value corresponds to empty
// string.
func toStr(s aws.StringValue) string {
	if s == nil {
		return ""
	}
	return *s
}

// toBool unpacks aws.BooleanValue to bool; nil value corresponds to false.
func toBool(b aws.BooleanValue) bool {
	if b == nil {
		return false
	}
	return *b
}

func stringMultiAZ(b bool) string {
	if !b {
		return ""
	}
	return "MultiAZ"
}

func stringVPC(b bool) string {
	if !b {
		return ""
	}
	return "VPC"
}
