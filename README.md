# r53-ddns
A DIY DynDNS service build upon AWS Route53

# Setup
This tool will try to determine the current external IPv4 and IPv6 address of
the host and then update an entry in a domain at Route53.  To do this you
obivously need a domain hosted at R53. If you already have a domain somewhere
else you can e.g. create a seperate subdomain like `dyn.yourdomain.com` which gets
delegated to R53 so you don't have to change anything in your current DNS setup
or migrate the domain.

# AWS costs
The costs you should be facing running this setup on AWS will be $0.50/month for
the hosted domain in R53 plus $0.4 per 1M requests. So you will pretty surely
never exceed $1 even after adding tax.

**WARNING**: Pushing just one wrong button in the AWS Console can cost you a lot
of money. So if you play around with cloud stuff, always configure a "Billing
alarm" so you wont be surprised at the end of the month by a big bill :)

# IAM Permissions
You need to create an IAM user which has the permission to update records on the
domain you want to use for DynDNS which should look like this:

```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": "route53:ChangeResourceRecordSets",
            "Resource": "arn:aws:route53:::hostedzone/<ZONE_ID>"
        }
    ]
}
```

Because this gives access to the complete zone, using a seperate subdomain like
desribed in the setup section is a good idea.

# Running this tool
Put the AWS keys you just created in the `~/.aws/credentials` file of the user
which will run this tool, or expose them via environment variables.
Then you can create a cronjob similar to this:
```
*/15 * * * * /usr/local/bin/r53-ddns -zone-id ABCDEF12345678 -domain dyn.example.com -hostname cray
```

If you want to set a record at the domain apex itself, use `-hostname @`.

