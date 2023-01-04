package metrics

import (
	"context"

	"github.com/Releem/mysqlconfigurer/releem-agent/config"
	"github.com/advantageous/go-logback/logging"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go/aws/awserr"
)

type AWSRDSInstanceGatherer struct {
	logger        logging.Logger
	debug         bool
	rdsclient     *rds.Client
	ec2client     *ec2.Client
	configuration *config.Config
}

func NewAWSRDSInstanceGatherer(logger logging.Logger, rdsclient *rds.Client, ec2client *ec2.Client, configuration *config.Config) *AWSRDSInstanceGatherer {

	if logger == nil {
		if configuration.Debug {
			logger = logging.NewSimpleDebugLogger("AWSRDSInstance")
		} else {
			logger = logging.NewSimpleLogger("AWSRDSInstance")
		}
	}

	return &AWSRDSInstanceGatherer{
		logger:        logger,
		debug:         configuration.Debug,
		rdsclient:     rdsclient,
		ec2client:     ec2client,
		configuration: configuration,
	}
}

func (awsrdsinstance *AWSRDSInstanceGatherer) GetMetrics(metrics *Metrics) error {

	//output := make(MetricGroupValue)
	info := make(MetricGroupValue)

	// Prepare request to RDS
	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &awsrdsinstance.configuration.AwsRDSDB,
	}

	// Request to RDS
	result, err := awsrdsinstance.rdsclient.DescribeDBInstances(context.TODO(), input)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			awsrdsinstance.logger.Error(aerr.Error())

		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			awsrdsinstance.logger.Error(err.Error())
		}
	} else {
		awsrdsinstance.logger.Println("RDS.DescribeDBInstances SUCCESS")

		// Prepare results
		for _, r := range result.DBInstances {

			awsrdsinstance.logger.Debug("DBInstance ", r.DBInstanceIdentifier)
			awsrdsinstance.logger.Debug("DBInstanceClass ", r.DBInstanceClass)
			awsrdsinstance.logger.Debug("ProcessorFeatures ", r.ProcessorFeatures)

			// // Prepare request to Ec2 to determine CPU count and Ram for InstanceClass
			// ec2input := &ec2.DescribeInstanceTypesInput{
			// 	InstanceTypes: []types.InstanceType{types.InstanceType(*r.DBInstanceClass)},
			// }

			// // Request to EC2 to get Type info
			// typeinfo, infoerr := awsrdsinstance.ec2client.DescribeInstanceTypes(context.TODO(), ec2input)

			// if infoerr != nil {
			// 	if aerr, ok := infoerr.(awserr.Error); ok {
			// 		awsrdsinstance.logger.Error(aerr.Error())

			// 	} else {
			// 		// Print the error, cast err to awserr.Error to get the Code and
			// 		// Message from an error.
			// 		awsrdsinstance.logger.Error(infoerr.Error())
			// 	}
			// } else {
			// 	awsrdsinstance.logger.Println("EC2.DescribeInstanceType SUCCESS")
			// 	awsrdsinstance.logger.Println("EC2.DescribeInstanceType %v", typeinfo)
			// }

			// for _, t := range typeinfo.InstanceTypes {
			// 	awsrdsinstance.logger.Debugf("ProcessorFeatures %v", t)
			// }

			if len(r.ProcessorFeatures) > 0 {
				for _, p := range r.ProcessorFeatures {
					awsrdsinstance.logger.Debug("Metric ", *p.Name, " has a value ", *p.Value)
				}
			}

			awsrdsinstance.configuration.Hostname = *result.DBInstances[0].DBInstanceIdentifier //output["DBInstance"] = *result.DBInstances[0].DBInstanceIdentifier
			info["CPU"] = MetricGroupValue{"Counts": 5}                                         //output["vNumCores"] = "5"
			//memory on byte
			info["PhysicalMemory"] = MetricGroupValue{"total": "8594128896"} //  output["TotalMemory"] = "8196"
			info["PhysicalMemory"] = MapJoin(info["PhysicalMemory"].(MetricGroupValue), MetricGroupValue{"swapTotal": "1074266112"})
			info["Host"] = MetricGroupValue{"InstanceType": "aws/rds"}
		}

	}

	metrics.System.Info = info
	awsrdsinstance.logger.Debug("collectMetrics ", metrics.System)
	return nil

}
