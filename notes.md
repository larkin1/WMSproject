# Program Notes/Plan

## General
 - Use supabase for data storage and api calls. This uses postgres and is reliable AF
 - store commits on the server in an event chain and total the numbers at export time (allows audits of commit history)
 - store a small database of items and their locations on the client side (updates periodically when in range of wifi)
 - on device add commits to a 'commit queue' in order to prevent mishaps with sketchy wifi
 - admin interface allows for reviewing of commits (timestamps, ids, etc), but commits are fixed (no modifying from anywhere)
 - admin interface allows for exporting of data and modification of data (via new commits, NOT modification of exitsting commits)
 - each commit gets a uuid4 to ensure we arent committing changes multiple times
 - maybe a device id for each commit, so a blame function can be made.

## How it will work
 - Sign in screen (for security)
 - User scans barcode
 - PDA opens a local copy of the database, and tries to sync the current location's data.
 - If the sync fails, display what item should be in that location,
 - Else, if the sync succeeds, the pda shows the item, and the count for that location.
 - A box is shown, with an option to either ADD or SUBTRACT items from the location
 - Once the user enters the data and presses a "commit" button,
   the data is gathered into a json file. In the json is:
   - the timestamp that the "commit" button was pressed at
   - a uuid4 for the id of the commit
   - the amount to modify the value by (in a signed int)
   - the location ID
   - the device ID
   - the item ID
 - the data is then sent to a queue to be committed at the soonest possible time
   (the soonest time being when wifi is available)
 - then the user is taken back to the scan screen to keep moving.

---

 - An adimin console will also be available, details to be confirmed.

## Setup
 - input the url for the database
 - input the id/name of the PDA
 - sync the database to the PDA while connected to wifi
 - Done!



