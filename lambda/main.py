from google.cloud import storage
import requests 

github_token = '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'
github_url = 'https://api.github.com/repos/cecillyaliu/webapp/zipball/main'
gcs_bucket_name = 'csye6225-dev'
source_file_name = "output.zip"

def downloadZip():
    headers = {
        "Authorization" :  'Bearer ' + github_token,
        "Accept": 'application/vnd.github.v3+json'
    }

    OWNER = 'cecillyaliu'
    REPO  = 'webapp'

    REF  = 'main'  # branch name

    EXT  = 'zip'
    url = f'https://api.github.com/repos/{OWNER}/{REPO}/{EXT}ball/{REF}'

    print('url:', url)

    r = requests.get(url, headers=headers)

    if r.status_code == 200:
        print('size:', len(r.content))
        with open(f'output.{EXT}', 'wb') as fh:
            fh.write(r.content)
        print(r.content[:10])
    else:
        print(r.text)
    source_file_name = "output.zip"   
    print("python main function")


def storeGCSBucket(bucket_name, source_file_name, destination_blob_name):
    """Uploads a file to the bucket."""

    storage_client = storage.Client()
    bucket = storage_client.bucket(bucket_name)
    blob = bucket.blob(destination_blob_name)

    generation_match_precondition = 0

    blob.upload_from_filename(source_file_name, if_generation_match=generation_match_precondition)

    print(
        f"File {source_file_name} uploaded to {destination_blob_name}."
    )

def sendEmail(email, subject, body):
    # Initialize the SES client
    ses_client = boto3.client('ses', region_name='us-west-2')

    # Specify the sender and recipient email addresses
    sender_email = 'cecillyaliu@gmail.com'
    recipient_email = email

    # Specify the email content
    email_content = {
        'Subject': {'Data': subject},
        'Body': {'Text': {'Data': body}}
    }

    # Send the email
    ses_client.send_email(
        Source=sender_email,
        Destination={'ToAddresses': [recipient_email]},
        Message=email_content
    )


def writeToDynamoDB(receiver, last_name, first_name):
    # Initialize the DynamoDB client
    dynamodb_client = boto3.client('dynamodb', region_name='your-aws-region')

    # Specify the DynamoDB table name
    table_name = 'dynamo-csye6225'

    # Specify the item to be written to DynamoDB
    dynamodb_item = {
        'Receiver': {'S': sender},
        'Subject': {'S': subject}
    }

    # Put item into DynamoDB table
    dynamodb_client.put_item(
        TableName=table_name,
        Item=dynamodb_item
    )



def handler(event, context):
    downloadZip()
    storeGCSBucket(gcs_bucket_name, source_file_name, source_file_name)
    sendEmail()
    writeToDynamoDB(event.user_info.email,event.user_info.last_name,event.user_info.first_name)


# if __name__ == '__main__':
#     main()
