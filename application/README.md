# webapp
Prerequisites:
Using GO, gorm

Build and Deploy:
Start up Sql workbench and then run the application

DEMO certificate:
[root@ip-172-31-30-32 demo.cecilialiu.cc]# openssl req -newkey rsa:2048 -new -nodes -x509 -days 365 -keyout demo.cecilialiu.cc.key.pem -out demo.cecilialiu.cc.cert.pem
...................+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++*..+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++*.................+.....+.+...+.....+....+..+.+.....+.......+...+...+.....................+.....+....+..+...+.......+..+.+............+..+.......+......+..+.......+.....+......+........................+...............+...+.............+........+....+...........+.+.....+.+..+.................................+.............+...........+................+..+.+..+............+............+......+....+.....+...+...+...............+...+.......+..+..........+..+...+...+..........+........+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
..............+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++*...+......+.....+.........+...+...+.+......+...+............+..+...+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++*...............+.........+...+...........................+...+....+...+..+....+......+...............+...+........+...+....+......+..+..........+..+.+..+.............+..+....+.........+..+...+....+......+......+.........+..+...+................+.........+..+...+....+..+...............+....+.....+.......+..+..........+...+........+.......+..+...+....+......+........+..........+.....+.........+.+.........+..+......+....+........+..........+..+....+......+.....+....+..+....+...+..+.+........+.+......+.....+............+...+....+........+.+.........+...+.....+.......+..+.........+.........+......+......+.+.....+....+.........+..+.......+...+..+...+............+...+.......+..........................+..........+..+....+.....+...............+.+............+.....+.+...........+...+......+.+...+.....+.+......+......+..+.+.....................+...+..+.............+...........+......+...+..........+........+...+..........+.....+.......+...+.....+...............+....+...+..+.......+...+..+................+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
-----
You are about to be asked to enter information that will be incorporated
into your certificate request.
What you are about to enter is what is called a Distinguished Name or a DN.
There are quite a few fields but you can leave some blank
For some fields there will be a default value,
If you enter '.', the field will be left blank.
-----
Country Name (2 letter code) [XX]:us
State or Province Name (full name) []:wa
Locality Name (eg, city) [Default City]:seattle
Organization Name (eg, company) [Default Company Ltd]:NE
Organizational Unit Name (eg, section) []:a10
Common Name (eg, your name or your server's hostname) []:demo.cecilialiu.cc
Email Address []:cecillyaliu@gmail.com
[root@ip-172-31-30-32 demo.cecilialiu.cc]# ll
total 8
-rw-r--r--. 1 root root 1436 Dec  6 02:45 demo.cecilialiu.cc.cert.pem
-rw-------. 1 root root 1704 Dec  6 02:44 demo.cecilialiu.cc.key.pem
[root@ip-172-31-30-32 demo.cecilialiu.cc]# aws acm import-certificate --certificate fileb://demo.cecilialiu.cc.cert.pem --private-key fileb://demo.cecilialiu.cc.key.pem --tags Key=Name,Value=Demo-CA-TEST
{
"CertificateArn": "arn:aws:acm:us-west-2:407091433811:certificate/9139f429-8430-4fb9-bf45-fa95bb693150"
}
[root@ip-172-31-30-32 demo.cecilialiu.cc]#