# CloudWatch Logs Usage Guide

This guide shows how to use AWS CLI with Gonzo for monitoring CloudWatch Logs.

## Prerequisites

### Installing AWS CLI

Get the AWS CLI from the official AWS documentation: [Installing or updating the latest version of the AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)

### Get Your Log Groups

A handy way to get your Log groups for your configured AWS region is:

```bash
aws logs describe-log-groups \                                                                                                                                                              
  --query 'logGroups[*].{Name:logGroupName,ARN:arn}' \           
  --output json   
```

## Using AWS CLI Tail with Gonzo

For detailed information about the AWS CLI logs tail command, see the [AWS CLI logs tail reference](https://awscli.amazonaws.com/v2/documentation/api/latest/reference/logs/tail.html).

### Basic Usage with Gonzo

The simplest way to use Gonzo with CloudWatch Logs is to pipe the output of `aws logs tail` directly to Gonzo:

```bash
aws logs tail [log_group_name] --follow | gonzo
```

For example:

```bash
aws logs tail "/aws/eks/stage/cluster" --follow | gonzo
```

**Note:** "detailed" output format of one log per line (non JSON) is the default for this command in AWS CLI. You can output JSON with the following command (to gain attribute visibility in Gonzo), although each JSON log message spans two lines for each message:

```bash
aws logs tail "/aws/eks/stage/cluster" --follow --format json | gonzo
```

### Multiple Log Groups

The basic `logs tail` command takes only one log group as input. You can use something like the following to tail logs into Gonzo from multiple log groups:

```bash
gonzo - < <(                                                                                                                                                                                    
  aws logs tail "/aws/eks/stage/cluster" --follow --format json &
  aws logs tail "RDS" --follow --format json &
  wait
)
```

This approach:
- Starts multiple `aws logs tail` processes in the background (`&`)
- Combines their output streams
- Pipes the combined output to Gonzo
- Uses `wait` to ensure all background processes complete properly

## Using AWS CLI Live Tail with Gonzo

Live Tail is a newer capability in the AWS CLI that provides a near real-time streaming of log events as they are ingested into selected log groups.

For detailed information about the AWS CLI start-live-tail command, see the [AWS CLI start-live-tail reference](https://awscli.amazonaws.com/v2/documentation/api/latest/reference/logs/start-live-tail.html).

### Basic Usage with Gonzo

```bash
aws logs start-live-tail --log-group-identifiers [ARN] | gonzo
```

ARNs for multiple log groups can also be provided, e.g.:

```bash
aws logs start-live-tail --log-group-identifiers "arn:aws:logs:us-east-1:767397775588:log-group:/aws/eks/stage/cluster" "arn:aws:logs:us-east-1:767397775588:log-group:RDS" | gonzo
```

The live tail output defaults to JSON, which ensures log attributes are captured by Gonzo.

## Important Cost Considerations

**Note:** AWS costs may be incurred for using the AWS CLI `tail` or `start-live-tail` commands. Please consult the [AWS CloudWatch Logs pricing documentation](https://aws.amazon.com/cloudwatch/pricing/) for details on potential charges related to log data retrieval for tail or live tail usage.
