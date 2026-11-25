# Core

Who uses the platform and what they own: organizations, users, and the admin. Every other
context refers to an organization.

## Language

**Organization**:
A client company that uses the platform. It owns its devices, device models, device groups,
and integrations. No other organization sees them.
_Avoid_: account, company, client, customer, tenant

**Deactivated organization**:
An organization whose users cannot log in. Its integrations still deliver uplinks, so the
admin can reactivate it with no lost data.
_Avoid_: suspended, blocked, disabled, deleted

**User**:
A person who logs in. A user is a member of one organization, or is an admin.
_Avoid_: account, login, operator

**Admin**:
A user who operates the platform. An admin is a member of no organization. An admin reads the
data of all organizations and changes none of it.
_Avoid_: superuser, root, operator

**Temporary password**:
A password that the platform generates when the admin creates or resets a user. The user must
replace it at the next login before doing anything else.
_Avoid_: initial password, one-time password (that name means a 2FA code)

**Profile**:
The contact data of a user: first name, last name, email, phone number, and UI language. The
user fills it in, not the admin.
_Avoid_: user details, contact card
