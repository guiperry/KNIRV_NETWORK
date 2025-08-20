module.exports = {
  extends: 'next/core-web-vitals',
  overrides: [
    {
      files: ['*.cjs'],
      rules: {
        '@typescript-eslint/no-require-imports': 'off'
      }
    }
  ]
}