/**
 * ts-jest AST transformer: Jest runs the arena as CommonJS, where
 * `import.meta` is a syntax error. Vite code reads `import.meta.env` and
 * `import.meta.url`; rewrite `import.meta` to an equivalent object so modules
 * keep using the native Vite API in production builds.
 *
 *   import.meta  →  ({ env: process.env, url: require('url').pathToFileURL(__filename).href })
 */
const name = 'knirvarena-import-meta';
const version = 1;

function factory(tsCompiler) {
  const ts = tsCompiler.configSet.compilerModule;
  return (ctx) => {
    const f = ctx.factory;
    const replacement = () =>
      f.createParenthesizedExpression(
        f.createObjectLiteralExpression([
          f.createPropertyAssignment('env', f.createPropertyAccessExpression(f.createIdentifier('process'), 'env')),
          f.createPropertyAssignment(
            'url',
            f.createPropertyAccessExpression(
              f.createCallExpression(
                f.createPropertyAccessExpression(
                  f.createCallExpression(f.createIdentifier('require'), undefined, [f.createStringLiteral('url')]),
                  'pathToFileURL'
                ),
                undefined,
                [f.createIdentifier('__filename')]
              ),
              'href'
            )
          ),
        ])
      );
    const visit = (node) => {
      if (ts.isMetaProperty(node) && node.keywordToken === ts.SyntaxKind.ImportKeyword && node.name.text === 'meta') {
        return replacement();
      }
      return ts.visitEachChild(node, visit, ctx);
    };
    return (sourceFile) => ts.visitNode(sourceFile, visit);
  };
}

module.exports = { name, version, factory };
