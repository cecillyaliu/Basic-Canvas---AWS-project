import os
import json
import uuid
import logging
import requests
from datetime import datetime
import boto3
from botocore.exceptions import ClientError
from google.cloud import storage
from google.oauth2 import service_account


# set up the env var
AWS_REGION = "us-west-2"
github_token = os.environ['github_token']
gcs_sa_token = os.environ['google_token']
RECIPIENT = os.environ['RECIPIENT']
BUCKET_NAME = os.environ['BUCKET_NAME']
DynamoDb_Table = os.environ['DYNAMODB_TABLE_NAME']

os.environ["GOOGLE_APPLICATION_CREDENTIALS"]="gcp-secret.json"

# Set up our logger
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger()


def lambda_handler(event, context):
    url = event['Records'][0]['Sns']['MessageAttributes']['url']['Value']
    print(url)
    
    # Replace recipient@example.com with a "To" address. If your account 
    # is still in the sandbox, this address must be verified.
    RECIPIENT = "success@simulator.amazonses.com"

    # bucket_name="csye6225-dev"
    source_file_name="/tmp/output.zip"
    destination_blob_name="output_" + str(uuid.uuid1()) + ".zip"
    
    download_zip_github(url)
    upload_zip_gcs(BUCKET_NAME, source_file_name, destination_blob_name)
    send_email_to_user(RECIPIENT)


def download_zip_github(url):
    headers = {
        "Authorization" : 'token ' + github_token,
        "Accept": 'application/vnd.github.v3+json'
    #    "Accept": '*.*',
    }

    REPO  = 'output'

    #REF  = 'main'  # branch name
    EXT  = 'zip'
    #EXT  = 'tar'  # it also works

    # url = f'https://api.github.com/repos/{OWNER}/{REPO}/{EXT}ball/{REF}'
    print('url:', url)

    r = requests.get(url, headers=headers)

    if r.status_code == 200:
        print('size:', len(r.content))
        # if runtime is aws lambda
        with open(f'/tmp/{REPO}.{EXT}', 'wb') as fh:
            fh.write(r.content)
        print(r.content[:10])  # display only some part
        logger.info(
            "Successfully download the zip package from  %s.", 
            url
        )
    else:
        print(r.text)
        logger.error(
            "Failed to download the zip package from  %s.", 
            url
        )


def upload_zip_gcs(bucket_name, source_file_name, destination_blob_name):
    """Uploads a file to the bucket."""

    try:
        storage_client = storage.Client()
        
        bucket = storage_client.bucket(bucket_name)
        blob = bucket.blob(destination_blob_name)

        generation_match_precondition = 0

        blob.upload_from_filename(source_file_name, if_generation_match=generation_match_precondition)

        print(
            f"File {source_file_name} uploaded to gs://{bucket_name}/{destination_blob_name}."
        )
    except ClientError as e:
        logger.error(
            f"File {source_file_name} was failed to upload to gs://{bucket_name}/{destination_blob_name}."
        )
    else:
        logger.info(
            f"File {source_file_name} was successfully uploaded to gs://{bucket_name}/{destination_blob_name}."
        )


def send_email_to_user(RECIPIENT):
    # Replace sender@example.com with your "From" address.
    # This address must be verified with Amazon SES.
    SENDER = "cecillyaliu@gmail.com"

    # Specify a configuration set. If you do not want to use a configuration
    # set, comment the following variable, and the 
    # ConfigurationSetName=CONFIGURATION_SET argument below.
    CONFIGURATION_SET = "my-first-configuration-set"

    # The subject line for the email.
    SUBJECT = "Amazon SES Test (SDK for Python)"

    # The email body for recipients with non-HTML email clients.
    BODY_TEXT = ("Amazon SES Test (Python)\r\n"
                "This email was sent with Amazon SES using the "
                "AWS SDK for Python (Boto)."
                )
                
    # The HTML body of the email.
    BODY_HTML = """<html>
    <head></head>
    <body>
    <h1>Amazon SES Test (SDK for Python)</h1>
    <p>This email was sent with
        <a href=' '>Amazon SES</a > using the
        <a href='https://aws.amazon.com/sdk-for-python/'>
        AWS SDK for Python (Boto)</a >.</p >
    </body>
    </html>
                """            

    # The character encoding for the email.
    CHARSET = "UTF-8"

    # Create a new SES resource and specify a region.
    client = boto3.client('ses',region_name=AWS_REGION)

    # Try to send the email.
    try:
        #Provide the contents of the email.
        response = client.send_email(
            Destination={
                'ToAddresses': [
                    RECIPIENT,
                ],
            },
            Message={
                'Body': {
                    'Html': {
                        'Charset': CHARSET,
                        'Data': BODY_HTML,
                    },
                    'Text': {
                        'Charset': CHARSET,
                        'Data': BODY_TEXT,
                    },
                },
                'Subject': {
                    'Charset': CHARSET,
                    'Data': SUBJECT,
                },
            },
            Source=SENDER,
            # If you are not using a configuration set, comment or delete the
            # following line
            ConfigurationSetName=CONFIGURATION_SET,
        )
    # Display an error if something goes wrong. 
    except ClientError as e:
        print(e.response['Error']['Message'])
        logger.error(
            "Email sent failed due to:  %s.", 
            e.response['Error']['Message']
        )
        update_mail_status_dynamodb(RECIPIENT, SUBJECT, 'Failed')
    else:
        print("Email sent! Message ID:"),
        print(response['MessageId'])
        logger.info(
            "Email sent! Message ID:  %s.", 
            response['MessageId']
        )
        update_mail_status_dynamodb(RECIPIENT, SUBJECT, 'Successful')

def update_mail_status_dynamodb(recipient, subject, sendStatus):
    dynamodb_client = boto3.resource("dynamodb",region_name=AWS_REGION)
    email_status_table = dynamodb_client.Table(DynamoDb_Table)

    try:
        email_status_table.put_item(
            Item={
                "id": str(uuid.uuid1()),
                "date_time" : str(datetime.now()),
                "recipient": recipient,
                "subject": subject,
                "sendStatus": sendStatus
            }
        )
        logger.info(
            "Email status %s %s was put to DynamoDB table successfully",
            recipient,
            subject,
        )
    except ClientError as err:
        logger.error(
            "Couldn't add mail status %s to table %s. Here's why: %s: %s",
            recipient,
            email_status_table.name,
            err.response["Error"]["Code"],
            err.response["Error"]["Message"],
        )
        raise
