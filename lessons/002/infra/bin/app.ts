#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { EksV2ClusterStack } from '../lib/eks-cluster';

const app = new cdk.App();
new EksV2ClusterStack(app, 'EksV2Stack', {
  env: { 
    account: process.env.CDK_DEFAULT_ACCOUNT, 
    region: process.env.CDK_DEFAULT_REGION 
  },
});

app.synth();
