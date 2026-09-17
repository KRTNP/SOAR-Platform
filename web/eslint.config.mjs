import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";
import jsxA11y from "eslint-plugin-jsx-a11y";

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,

  // eslint-config-next enables only a handful of jsx-a11y rules (alt-text,
  // aria-props, aria-proptypes, aria-unsupported-elements, role-has-required-
  // aria-props, role-supports-aria-props). This console is operated by one
  // person deciding whether to let an action touch a firewall, sometimes from
  // the keyboard, and the accessibility defects that matter here — a control
  // that is a div, a label with nothing associated to it, a role applied to an
  // element that cannot carry it — are all outside that subset. Placed after
  // the Next configs so the full set wins over the partial one.
  //
  // Scope note: these rules are per-element. They cannot tell whether an
  // `aria-labelledby` actually resolves to an id that exists elsewhere in the
  // tree, so that class of bug still needs a human or a rendered page.
  // Rules only, not the whole flat config: eslint-config-next already
  // registers the `jsx-a11y` plugin, and re-registering it is a hard
  // ConfigError ("Cannot redefine plugin"). Borrowing just the rule map
  // upgrades the existing namespace from the partial set to all 33.
  {
    name: "soar/jsx-a11y-strict",
    rules: {
      ...jsxA11y.flatConfigs.strict.rules,

      // Deprecated upstream and superseded by label-has-associated-control,
      // which strict also enables. Keeping both means the dead rule reports
      // on labels the live rule is satisfied with.
      "jsx-a11y/label-has-for": "off",

      // autoFocus is correct in exactly one place here and wrong everywhere
      // else. The ARIA dialog pattern requires focus to move *into* a dialog
      // when it opens, and components/Modal.tsx implements that by preferring
      // `[autofocus], [data-autofocus]` inside the panel — the closure-reason
      // and delete-confirm inputs carry it so the operator can start typing
      // the reason they are required to give. The rule's actual target is
      // autofocus on page load, which it cannot distinguish from this. Left
      // off with that reasoning recorded rather than sprinkling
      // eslint-disable comments across the dialogs.
      "jsx-a11y/no-autofocus": "off",
    },
  },

  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
  ]),
]);

export default eslintConfig;
