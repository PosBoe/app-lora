# Core

Who uses the platform and what they own: organizations, users, and the admin. Every other
context refers to an organization.

## Language

**Organization**:
A client company that uses the platform. It owns its devices, device models, device groups,
and integrations. No other organization sees them.
_Avoid_: account, company, client, customer, tenant

**User**:
A person who logs in. A user is a member of one organization, or is an admin.
_Avoid_: account, login, operator

**Admin**:
A user who operates the platform. An admin is a member of no organization and sees all
organizations.
_Avoid_: superuser, root, operator

**Profile**:
The contact data of a user: first name, last name, email, and phone number. The user fills it
in, not the admin.
_Avoid_: user details, contact card
