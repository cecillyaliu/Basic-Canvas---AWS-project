sudo apt-get update
sudo apt-get upgrade -y
sudo apt-get clean

sudo mv /home/admin/webapp.service /etc/systemd/system
sudo systemctl daemon-reload
sudo systemctl enable webapp.service

sudo mkdir /home/admin/env


#A7
echo "Installing AWS CloudWatch Agent"
#install th cloudwatch agent
sudo apt-get update
wget https://amazoncloudwatch-agent.s3.amazonaws.com/debian/amd64/latest/amazon-cloudwatch-agent.deb
sudo dpkg -i -E ./amazon-cloudwatch-agent.deb


#copy the cloudwatch-config.json file
sudo cp /home/admin/cloudwatch-config.json /opt/cloudwatch-config.json

# Start the CloudWatch Agent
sudo /opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl -a fetch-config -m ec2 -s -c file:/opt/cloudwatch-config.json

mkdir -p /home/admin/webservice/logs
touch /home/admin/webservice/logs/csye6225.log
chmod 775 /home/admin/webservice/logs/csye6225.log

# Enable and start the agent
sudo systemctl enable amazon-cloudwatch-agent
sudo systemctl start amazon-cloudwatch-agent