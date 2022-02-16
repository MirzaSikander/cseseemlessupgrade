Small prototype to show that you can update the CSE without causing a reimage or even a reboot. 


Set up environment variables.
```
export AZURE_CLIENT_ID="__CLIENT_ID__"
export AZURE_CLIENT_SECRET="__CLIENT_SECRET__"
export AZURE_TENANT_ID="__TENANT_ID__"
export AZURE_SUBSCRIPTION_ID="__SUBSCRIPTION_ID__"
export MS_ALIAS="__MS_ALIAS__"
```

Use `./create/main.go` to create the resources. It will provision an rg, virtual network, vmss with one instance, and other resources for bastion. At the end it will output the username and password to login into the VM.

Once you have logged in, you can use `./addextension/main.go` to add the extension. The CSE is very simple. It just creates a file with the current time stamp in `/tmp`.

To rerun/update the extension, change the `version` in `./addextension/main.go` and run it again.

At the end, you will see something like this.

```
azureuser@vmss000000:~$ ls /tmp
log-02-16-2022-04:38:06.log  systemd-private-278125bb9e79404ebddafe8080482c26-systemd-resolved.service-hlvbs5
log-02-16-2022-04:42:03.log  systemd-private-278125bb9e79404ebddafe8080482c26-systemd-timesyncd.service-cAP9Xv
azureuser@vmss000000:~$ last reboot
reboot   system boot  5.4.0-1067-azure Wed Feb 16 03:53   still running

wtmp begins Wed Feb 16 03:53:55 2022
```