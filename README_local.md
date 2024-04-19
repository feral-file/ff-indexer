## This doccument repesents steps to run indexer services locally.

# Have you AWS account created, make some initial setup:

change password, assign MFA, create Access key, grant your access to "DevelopmentEksStack01"
add config.yaml file for services you want to run

# Configure your account:

aws configure
input access key ID, access secret, region.

# configure the mfa device only first time

aws configure set default.mfa arn:aws:iam::083397868157:mfa/MY-AUTH

# look up up the auth number in auth app each time you need a new token

./aws-mfa-token <OTP>

# this command should now work until the token expires

WS_PROFILE=session aws ssm get-parameter --name arn:aws:ssm:ap-northeast-1:<your_aws_id>:parameter/autonomy/development/jwtPublicKey

# you probably should export to save typing the env var each time

export AWS_PROFILE=session

# config Kubernetes

aws eks update-kubeconfig --name DevelopmentEksStack01 --region ap-northeast-1

# run service

from root folder:
cd services/<service_name>
go run .
you should see that your local server listening to port that you config

creates a secure tunnel between a local machine and a pod running in Kubernetes cluster. this should run parallelly with you server:
kubectl -n mongodb-sharded port-forward pod/mongodb-sharded-mongos-949bf8c8c-wjhst 27017:27017

# test your local server

curl -is http://localhost:8080/healthz
