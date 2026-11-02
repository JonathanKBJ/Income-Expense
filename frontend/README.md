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
