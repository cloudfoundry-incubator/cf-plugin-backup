# cf-plugin-backup
A Cloud Foundry Plugin that allows backup and restore of the CCDB using CF API.

## Limitations

   - The basic use case for the plugin is saving the state of a CF
     instance about to be modified in some way and then restoring the
     instance from scratch when encountering problems.

     The key words in the above sentence are __from scratch__.

     It is when trying to apply restore on a non-pristine CF instance
     that the limitations take hold:

     While application configuration is restored even for an
     (still-)existing application, this configuration is not reflected
     into the runtime when the application in question is (already,
     still) running.

     Restore does not compare old and new configuration to determine
     that the application should be restarted for the new
     configuration to take effect.

   - User information is managed by the UAA, not the Cloud Controller
     (CC). As the plugin talks only to the CC it cannot save full user
     information, nor restore users.

     Saving and restoring users has to be done separately, and
     restoration has to happen before the backup plugin is invoked.

   - The set of available stacks is part of the CF instance setup, not
     of the CC configuration. Attempting to restore applications using
     stacks not available on the target CF instance will fail.

     Setting the necessary stacks has to be done separately and before
     the backup plugin is invoked.

   - Buildpacks are not saved, neither standard, nor
     custom. Attempting to restore applications using buildpacks not
     available on the target CF instance will fail.

     Saving and restoring buildpacks has to be done separately, and
     restoration has to happen before the backup plugin is invoked.

## Install the Backup/restore plugin

1. Download the plugin from https://github.com/SUSE/cf-plugin-backup/releases

2. Install using cf install-plugin:
`cf install-plugin <backup-restore-binary> `

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

To back up all of the Cloud Application Platform data, including
applications, use this command to create a new backup snapshot to a
local file:
`cf backup-snapshot`

This will save your cloud foundry information into a file in your
current directory called `cf-backup.json`, and your application data
into a local subdirectory called `app-bits/`

The saved information contains:

   - Org Quota Definitions
   - Space Quota Definitions
   - Orgs
      - Spaces
         - Applications
         - Users references (role in the space)
      - (private) Domains
      - Users references (role in the org)
      - Routes
      - Route Mappings
      - Stack References
   - Shared Domains
   - Security Groups
   - Feature Flags
   - Application droplets (zip files holding the staged app)

Note how it does not save user information. Only the references needed
for the roles. The full user information is handled by the UAA, this
plugin talks only to the CC.

### Restore a previous Cloud Application Platform backup

To restore all of the Cloud Application Platform data, including
applications, navigate to the directory which contains your
`cf-backup.json` and `app-bits/` and run this command:
`cf backup-restore`

There are 2 optional parameters that can be used when restoring:

* `[--include-security-groups]`
* `[--include-quota-definitions]`

### View the current snapshot

To show you what information exists about the current backup, use this command:

`cf backup-info`

## Scope of the restore

Scope | Restore
---|---
Orgs\* | Yes
Org auditors | Yes
Org manager | Yes
Org billing-manager | Yes
Quota definitions | Optional `[--include-quota-definitions]`
Spaces* | Yes
Space Quotas | Optional `[--include-quota-definitions]`
Space developers | Yes
Space auditors | Yes
Space managers | Yes
Apps | Yes
App binaries | Yes
Routes | Yes
Route mappings | Yes
Domains | Yes
Private domains | Yes
Stacks | N/A
Feature flags | Yes
Security groups | Optional `[--include-security-groups]`
Custom buildpacks | No

*Organization and space users are backed up at the Cloud Application
Platform level. The user account in UAA/LDAP, as well as service
instances and their application bindings are not backed up.

Details:

   - Shared Domains: Attempts to create domains from the
     backup. Existing domains are retained, __and not overwritten__.

   - Feature Flags: Attempts to update flags from the backup.

   - Quota Definitions: Existing quotas are overwritten from the
     backup (deleted, re-created).

   - Orgs: Attempts to create orgs from the backup. Attempts to
     update existing orgs from the backup.

      - Space Quota Definitions: Existing quotas are overwritten from
        the backup (deleted, re-created).

      - User roles: Expects the referenced user to exist. Will fail
        when the user is already associated with the space, in the
        given role.

      - (private) Domains: Attempts to create domains from the
        backup. Existing domains are retained, __and not
        overwritten__.

      - Spaces: Attempts to create spaces from the backup. Attempts to
        update existing spaces from the backup.

         - User roles: Expects the referenced user to exist. Will fail
           when the user is already associated with the space, in the
           given role.

         - Apps: Attempts to create apps from the backup. Attempts to
           update existing apps from the backup (memory, instances,
           buildpack, state, ...)

   - Security groups: Existing groups are overwritten from the backup
     (deleted, re-created)
