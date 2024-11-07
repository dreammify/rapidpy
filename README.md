# rapidpy

TL;DR: Put a python file in applications and the service will try to make sure it runs. The file must start with `# 0` for an application intended to run continuously or `# <Int>` for an application intended to run once every `<Int>` seconds. Applications can only use the `requests` and `flask` python libraries and the standard library.
