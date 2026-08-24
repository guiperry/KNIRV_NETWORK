import React, { useState } from 'react';
import { GitBranch } from 'lucide-react';

const source = `function verifyToken(token) {
  if (!token || token.length < 20) {
    return false;
  }
  const parts = token.split('.');
  return parts.length === 3;
}`;

const treeText = `(program
  (function_declaration
    name: (identifier) ; "verifyToken"
    parameters: (formal_parameters
      (identifier)) ; "token"
    body: (statement_block
      (if_statement
        condition: (binary_expression
          left: (unary_expression (identifier))
          right: (binary_expression
            left: (member_expression
              object: (identifier)
              property: (property_identifier))
            right: (number)))
        consequence: (statement_block
          (return_statement (false))))
      (lexical_declaration
        (variable_declarator
          name: (identifier) ; "parts"
          value: (call_expression
            function: (member_expression)
            arguments: (arguments (string)))))
      (return_statement
        (binary_expression
          left: (member_expression)
          right: (number))))))`;

const queries = [
  { label: 'all function names', q: '(function_declaration name: (identifier) @fn.name)' },
  { label: 'string literals', q: '(string) @str' },
  { label: 'return statements', q: '(return_statement) @ret' },
  { label: 'binary comparisons', q: '(binary_expression) @cmp' },
];

const matches: Record<string, string[]> = {
  '(function_declaration name: (identifier) @fn.name)': ['@fn.name  1:9-1:20  "verifyToken"'],
  '(string) @str': ["@str  5:24-5:27  '.'"],
  '(return_statement) @ret': ['@ret  3:4-3:17  "return false;"', '@ret  7:2-7:29  "return parts.length === 3;"'],
  '(binary_expression) @cmp': ['@cmp  2:6-2:35  "!token || token.length < 20"', '@cmp  2:16-2:35  "token.length < 20"', '@cmp  7:9-7:28  "parts.length === 3"'],
};

const TreeSitter: React.FC = () => {
  const [query, setQuery] = useState(queries[0].q);

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-lime-500/20 rounded-lg">
          <GitBranch className="w-6 h-6 text-lime-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Tree-sitter</h1>
          <p className="text-slate-400 text-sm font-mono">parser: tree-sitter-javascript · incremental: enabled</p>
        </div>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-3">
          <div className="text-xs text-slate-600 uppercase mb-2">source.js</div>
          <pre className="text-xs font-mono text-slate-200 whitespace-pre-wrap leading-relaxed">{source}</pre>
        </div>

        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-3">
          <div className="text-xs text-slate-600 uppercase mb-2">Syntax tree</div>
          <pre className="text-xs font-mono text-lime-300/90 whitespace-pre leading-relaxed overflow-x-auto max-h-[420px] overflow-y-auto">{treeText}</pre>
        </div>

        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-3 flex flex-col">
          <div className="text-xs text-slate-600 uppercase mb-2">S-expression query</div>
          <div className="flex flex-wrap gap-1 mb-2">
            {queries.map(q => (
              <button
                key={q.label}
                onClick={() => setQuery(q.q)}
                className={`text-[10px] px-2 py-1 rounded font-mono border ${
                  query === q.q ? 'bg-lime-500/20 border-lime-500/40 text-lime-300' : 'border-slate-700/50 text-slate-500'
                }`}
              >
                {q.label}
              </button>
            ))}
          </div>
          <textarea
            value={query}
            onChange={e => setQuery(e.target.value)}
            spellCheck={false}
            className="w-full h-16 bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1.5 text-xs font-mono text-slate-200 resize-none mb-3"
          />
          <div className="text-xs text-slate-600 uppercase mb-1">Matches</div>
          <div className="flex-1 space-y-1 font-mono text-xs text-slate-300 overflow-y-auto">
            {(matches[query] ?? ['-- no cached matches for this query --']).map((m, i) => (
              <div key={i} className="bg-slate-900/60 rounded px-2 py-1">{m}</div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};

export default TreeSitter;
