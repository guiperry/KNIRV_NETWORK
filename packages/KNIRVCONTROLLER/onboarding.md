# KNIRV Controller Onboarding Storyboard

## Purpose

This document storyboards the onboarding welcome sequence for `KNIRVCONTROLLER`.
It reflects the current React app shape in `src/react-app/pages/Onboarding.tsx`,
and extends it to cover the platform entry flow used by `ONBOARDING.KNIRV.COM`.

Important context:

- The controller is not only a local vault.
- It is also the signing vault that proves identity during platform access.
- The onboarding flow therefore has two jobs:
  - establish the controller vault on the device
  - bind that vault to a platform session by signing an authentication challenge

## What The Current App Already Does

From the codebase, the current onboarding page already supports:

- Welcome screen with `Create New Vault` and `Import Existing Vault`
- Local password setup for the vault
- Mnemonic generation on vault creation
- Mnemonic import for existing vaults
- Success state with redirect into the app shell

The current vault hook persists the vault in `localStorage` and unlocks it with a password.
That means the controller is already acting as the local root identity for the device.

Implementation note:

- `Onboarding.tsx` currently redirects to `/vault`
- The router currently defines `/vault`
- The storyboard below assumes the intended destination is the vault area and the route should be normalized during implementation

## Storyboard Overview

The welcome sequence should feel like a secure handoff:

1. Introduce the controller as the root KNIRV identity
2. Let the user create or restore the controller vault
3. Ask the user to set a local device password
4. Reveal and confirm the recovery phrase
5. Use the controller vault to sign a platform challenge
6. Confirm platform access and open the main KNIRV surface

## Storyboard Frames

### Frame 1: Entry Gate

- Screen: `ONBOARDING.KNIRV.COM`
- Visual: Dark glass panel, blue accent glow, shield emblem, minimal motion in the background
- Headline: `Welcome to KNIRV`
- Subtext: `Your controller vault is the key to the platform`
- Primary action: `Create Controller`
- Secondary action: `Restore Controller`

Goal:

- Make it clear that this is not a generic signup page
- Establish that the controller vault is the root access credential

### Frame 2: Identity Choice

- Screen: same shell, two large cards
- Card 1: `Create New Controller`
  - Description: `Generate a fresh controller vault and recovery phrase`
- Card 2: `Import Existing Controller`
  - Description: `Restore a controller vault from a 12 or 24 word phrase`

Interaction:

- User chooses whether this device is creating the root vault or restoring one
- The choice should be presented as identity recovery, not account creation

### Frame 3: Device Password

- Screen: password form with confirm field
- Headline: `Set Device Password`
- Subtext: `This unlocks the controller on this device only`

Interaction:

- User creates a local password for vault unlock
- Password is used for device convenience and local protection
- This is separate from the recovery phrase and should not replace it

Behavior:

- Reject passwords shorter than the minimum policy
- Reject mismatched confirmation fields
- Keep the error language simple and direct

### Frame 4: Recovery Phrase Reveal

- Screen: phrase reveal panel with numbered words
- Headline: `Secret Recovery Phrase`
- Subtext: `Write these words down in order. Anyone with this phrase controls the vault`

Interaction:

- User copies or writes the phrase offline
- The UI should require an explicit acknowledgment before continuing

Visual treatment:

- Use a numbered grid
- Highlight each word as a discrete object
- Keep the warning visually stronger than the phrase itself

Security note:

- Do not reframe the phrase as a convenience feature
- Treat it as the master recovery mechanism for the controller

### Frame 5: Confirmation And Transition

- Screen: confirmation state
- Headline: `Controller Ready`
- Subtext: `Preparing secure platform access`

Interaction:

- User confirms the phrase was saved
- App transitions into the onboarding auth step

Goal:

- Move from vault creation to platform access without implying the process is complete

### Frame 6: Platform Signing Step

- Screen: signing prompt tied to the platform session
- Headline: `Sign To Enter KNIRV`
- Subtext: `The controller vault must sign a challenge to unlock platform access`

Interaction:

- Platform issues a nonce or challenge message
- Controller vault signs the message
- Backend verifies the signature and binds the session to the vault address

Suggested message pattern:

- `Authorize KNIRV Controller access`
- Include a nonce and timestamp
- Make the message human-readable so the user can inspect what they are signing

Behavior:

- If signature verification fails, the user stays outside the platform
- If verification succeeds, issue the authenticated session and continue

Security note:

- This step should feel distinct from recovery phrase handling
- Never ask for the recovery phrase during platform sign-in

### Frame 7: Access Granted

- Screen: success state with subtle pulse animation
- Headline: `Welcome Back, Controller`
- Subtext: `Your KNIRV session is active`

Interaction:

- Session is established
- User is routed into the main controller surface

Primary destination:

- Vault overview
- DVE list or platform home

Behavior:

- Show a short loading transition
- Then open the main application shell with vault status visible

## Recommended Welcome Copy

The welcome copy should be short and operational:

- `Welcome to KNIRV`
- `Your controller vault is the key to the platform`
- `Set a device password`
- `Save your recovery phrase`
- `Sign to enter KNIRV`
- `Vault ready`

Avoid:

- overexplaining blockchain mechanics on the first screen
- mixing recovery phrase language with sign-in language
- calling the controller a passive account; it is the root identity

## UX Notes

- Keep the sequence linear and non-branchy
- Use the same visual language as the rest of the app: dark glass panels, blue accents, restrained motion
- Make the vault status explicit at every stage
- Treat platform access as a privilege granted by vault ownership, not by email or password alone
- If the controller already exists, fast-path directly into unlock and signing

## Suggested Final Flow

1. Land on the welcome screen
2. Choose create or restore
3. Set local password
4. Back up recovery phrase
5. Confirm backup
6. Sign the platform challenge
7. Enter the main KNIRV interface

## Implementation Gaps To Resolve

- Normalize the post-onboarding redirect route
- Add an explicit signing challenge step for `ONBOARDING.KNIRV.COM`
- Decide whether the signing prompt lives in the app, the website, or both
- Make the controller-vault role visible in the copy so users understand it is the platform key
