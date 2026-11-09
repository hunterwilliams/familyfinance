# Family Finance iOS App

Opt-in receipt capture companion for the Family Finance API. **Nothing is imported automatically.**

## Features

- **Photos picker** — select a receipt screenshot or notification image
- **Share Extension** — share from SCB, Grab, LINE Pay, etc. (skeleton in `ShareExtension/`)
- **Vision OCR** — suggests amount, date, merchant (user always confirms)
- **Offline drafts** — SwiftData queue; sync when API is reachable

## Setup

1. Install [XcodeGen](https://github.com/yonaskolb/XcodeGen): `brew install xcodegen`
2. Generate Xcode project:
   ```bash
   cd ios
   xcodegen generate
   open FamilyFinance.xcodeproj
   ```
3. In the app (or via UserDefaults), set:
   - `apiBaseURL` — e.g. `https://your-coolify-domain.com`
   - `apiKey` — your production API key

4. For local dev on simulator, use your Mac's LAN IP (not `localhost`):
   - `http://192.168.x.x:8080`

## Capture flow

1. User screenshots a payment receipt or selects existing photo
2. OCR prefills amount / date / description
3. User picks family member, account, category
4. User taps **Save** → `POST /api/v1/transactions`

## Share Extension

Build the `ShareExtension` target. User taps Share on any receipt image → extension opens → same confirm flow.

## Privacy

- OCR runs on-device via Apple Vision
- No background photo scanning
- No access to other apps' notifications
