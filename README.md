# cf-plugin-backup
A Cloud Foundry Plugin that allows backup and restore of the CCDB using CF API.

## Install the Backup/restore plugin

1. Extract the plugin:  
`tar zxvf PATH/TO/<file>.tgz `  
  
2. Install using the cf plugin:   
`cf install-plugin <extracted-file> `  
_Where `extracted-file` is the name of the extracted file._   

3. Upon running the command, you will see the following message:   

~~~~
**Attention: Plugins are binaries written by potentially untrusted authors. Install and use plugins
at your own risk.**  
Do you want to install the plugin cf-plugin-backup? (y or n)>y  
Installing plugin cf-plugin-backup...  
OK  
Plugin Backup v1.0.x successfully installed.  
~~~~

4. To verify that the plugin was installed successfully, run the `cf help` command and look what is listed under **`Commands offered by installed plugins`**:

~~~~
  backup-info    
  backup-restore    
  backup-snapshot   
~~~~

5. If you try running one of the commands and see the following message when trying to use the Backup/Restore plugin, it is not installed:
~~~~
`cf backup-info`  
'backup-info' is not a registered command. See 'cf help'  
~~~~

## Using the Backup/restore plugin

### Back up the current Cloud Application Platform  

To back up all of the Cloud Application Platform data, including applications, use this command to create a new backup snapshot to a local file:  
`cf backup-snapshot`  

### Restore a previous Cloud Application Platform backup  

To restore all of the Cloud Application Platform data, including applications, use this command to restore the CloudFoundry state from a backup created using the snapshot command:  
`cf backup-restore`  

There are 2 optional parameters that can be used when restoring:

* `[--include-security-groups]`   
* `[--include-quota-definitions]`  

### View the current snapshot

To show you what information exists about the current backup, use this command:

`cf backup-info`

## Scope of the restore
orgs​*	Yes
Org auditors​	Yes
Org manager​	Yes
Org billing-manager​	Yes
Quota definitions​	Yes
spaces​*	Yes
Space developers​	Yes
Space auditors​	Yes
Space managers​	Yes
apps​	Yes
App binaries​	Yes
routes​	Yes
Route mappings​	Yes
domains​	Yes
Private domains​	Yes
stacks​	N/A
Feature flags​	Yes
security groups​	Optional

*Organization and space users are backed up at the Cloud Application Platform level. The user account in UAA/LDAP is not backed up.
