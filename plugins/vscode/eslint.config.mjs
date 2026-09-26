// ESLint flat config: recommended + type-aware rules from typescript-eslint.
import js from "@eslint/js";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["out/**", "node_modules/**", "media/**", "scripts/**", "bin/**"] },
  js.configs.recommended,
  ...tseslint.configs.recommendedTypeChecked,
  {
    languageOptions: { parserOptions: { projectService: true, tsconfigRootDir: import.meta.dirname } },
    rules: {
      "@typescript-eslint/no-floating-promises": "error", // every promise is awaited or explicitly voided
      "@typescript-eslint/no-explicit-any": "warn",
      eqeqeq: "error",
    },
  },
);
