# React + TypeScript + Vite

This template provides a minimal setup to get React working in Vite with HMR and some ESLint rules.

Currently, two official plugins are available:

- [@vitejs/plugin-react](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react) uses [Oxc](https://oxc.rs)
- [@vitejs/plugin-react-swc](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react-swc) uses [SWC](https://swc.rs/)

## React Compiler

The React Compiler is not enabled on this template because of its impact on dev & build performances. To add it, see [this documentation](https://react.dev/learn/react-compiler/installation).

## Expanding the ESLint configuration

If you are developing a production application, we recommend updating the configuration to enable type-aware lint rules:

```js
export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      // Other configs...

      // Remove tseslint.configs.recommended and replace with this
      tseslint.configs.recommendedTypeChecked,
      // Alternatively, use this for stricter rules
      tseslint.configs.strictTypeChecked,
      // Optionally, add this for stylistic rules
      tseslint.configs.stylisticTypeChecked,

      // Other configs...
    ],
    languageOptions: {
      parserOptions: {
        project: ['./tsconfig.node.json', './tsconfig.app.json'],
        tsconfigRootDir: import.meta.dirname,
      },
      // other options...
    },
  },
])
```

You can also install [eslint-plugin-react-x](https://github.com/Rel1cx/eslint-react/tree/main/packages/plugins/eslint-plugin-react-x) and [eslint-plugin-react-dom](https://github.com/Rel1cx/eslint-react/tree/main/packages/plugins/eslint-plugin-react-dom) for React-specific lint rules:

```js
// eslint.config.js
import reactX from 'eslint-plugin-react-x'
import reactDom from 'eslint-plugin-react-dom'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      // Other configs...
      // Enable lint rules for React
      reactX.configs['recommended-typescript'],
      // Enable lint rules for React DOM
      reactDom.configs.recommended,
    ],
    languageOptions: {
      parserOptions: {
        project: ['./tsconfig.node.json', './tsconfig.app.json'],
        tsconfigRootDir: import.meta.dirname,
      },
      // other options...
    },
  },
])
```

## Credit statements

The credit account editor accepts an optional official statement: statement date, closing balance, actual minimum payment, and payment due date. Enter all four values together, or use **Remove statement** to return to estimated billing.

For the matching current billing cycle, the account shows the bank's original minimum and the remaining amount separately. Remaining due subtracts recorded payments through today, retains cents, and is not prorated by the selected payment date. A payment marked `FULL` or `MINIMUM` does not settle an official minimum unless its amount covers the remaining due. New purchases do not change an already-issued minimum.

The statement applies only to its billing cycle; the next cutoff returns to estimated amounts until a new statement is entered. Next-cycle projections remain estimates, not bank-issued amounts.

The API accepts nullable `statementDate`, `statementBalance`, `statementMinimumDue`, and `statementDueDate` fields when creating/updating accounts. Dates use `YYYY-MM-DD`. Updates that omit the fields preserve the statement; `clearStatement: true` removes it. Account details expose `usesStatementMinimum` so payment screens can distinguish actual billing from estimates.

When rebuilding an account from a statement, back up its card ledger, installment plans, and linked dashboard bills first. Represent the opening balance once and import each statement transaction once; the signed ledger total must equal the closing balance. Do not also add the closing balance as another charge. Refunds currently use `PAYMENT`/`CUSTOM`; label them explicitly rather than treating them as cash transfers. Review dates and amounts on linked pending dashboard bills, and do not remove unrelated paid financial history.

Statement cutoffs and payment-day comparisons use the Bangkok calendar (`UTC+7`). Marking a linked dashboard bill paid records the credit payment on the actual payment day, independently of that bill's scheduled due date. This allows early payments to settle the current minimum without treating genuinely future-dated card payments as already paid.

## Early installment settlement

In an active installment's actions, choose **Settle early**, select the actual closure date, and optionally record a note. Record the real bank payment separately first: settlement does not create another payment, adjust the account's current balance, or close the credit account.

Settlement retains the original contract amounts, schedule dates, notes, actual paid-term count, and transaction history. It sets the plan to `COMPLETED`, clears its remaining balance, and records `earlySettlementDate` and `earlySettlementNote`. The plan no longer reserves future credit or contributes future installment dues; closed-early rows remain visible without presenting the unpaid terms as payments made. This is a permanent, read-only state, including linked historical ledger entries.

The API is `POST /api/credit-accounts/{accountId}/installments/{installmentId}/settle-early` with `{ "settlementDate": "YYYY-MM-DD", "note": "optional" }`. It requires the existing editor/group/account permissions and an active plan with unpaid terms. The date must be a valid calendar date no later than today in Bangkok. It may precede the first scheduled installment: the plan's start date is not proof of when funds were disbursed. Invalid dates return 400, missing or inaccessible plans return 404, and repeated settlement or mutations of an early-settled plan return 409.

Already-created Dashboard bills are account-level aggregates, not individual-plan records. Review them separately; settlement never deletes or rewrites them automatically. Existing payment receipts and unrelated active plans are preserved. The backend migration adds nullable settlement-date and default-empty settlement-note columns to existing installment records; deploy the backend before the frontend.

## Install on Android

The frontend includes `public/manifest.webmanifest` with a stable app ID, root start URL/scope, and `display: standalone`. Vite copies it and the launcher icons to `dist/`; no PWA plugin or extra dependency is required.

After deploying the frontend over HTTPS:

1. Open the deployed site directly in Chrome, not inside another app's browser.
2. Use Chrome's menu → **Add to Home screen** → **Install** (some versions show **Install app** directly). Choose installation rather than creating a browser shortcut.
3. Launch **Expenses** from the new icon. Pages within this site's scope should open without Chrome's address/title toolbar. Android's status and navigation bars remain visible. External sites may still open with browser controls.

An old home-screen shortcut may keep opening in a browser. Remove only that shortcut and install again; do not clear browser/site storage or delete an existing installed app's data just to replace the shortcut. The app still needs an internet connection: no service worker, offline mode, or financial-data cache is added. An automatic install banner is not guaranteed.

The PNG launcher icons use the existing app-header card emblem on an opaque dark background. `public/app-icon.svg` is their source; rasterize it at 192×192 and 512×512 when updating them. Keep the emblem inside the central 80%-diameter safe circle because the 512px icon also supports Android's maskable/adaptive shapes.

Verification: run `npm run build`, then `npm run preview`. In Chrome DevTools → **Application → Manifest**, check that the manifest parses, the display mode is standalone, and both PNG icons load at their declared dimensions. Final home-screen launch behavior must also be checked on the target Android device after deployment.
