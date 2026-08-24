import React from 'react';

interface TableHeader {
  label: string;
  align?: 'left' | 'center' | 'right';
  className?: string;
}

interface DataTableProps<T> {
  headers: TableHeader[];
  data: T[];
  renderRow: (item: T, index: number) => React.ReactNode;
  className?: string;
}

function DataTable<T>({ 
  headers, 
  data, 
  renderRow,
  className = ''
}: DataTableProps<T>) {
  const getAlignmentClass = (align?: string) => {
    switch (align) {
      case 'center': return 'text-center';
      case 'right': return 'text-right';
      default: return 'text-left';
    }
  };

  return (
    <div className={`w-full overflow-x-auto ${className}`}>
      <table className="w-full border-collapse">
        <thead>
          <tr className="border-b border-slate-700/50">
            {headers.map((header, index) => (
              <th 
                key={index} 
                className={`
                  px-4 py-3 text-blue-400 font-medium
                  ${getAlignmentClass(header.align)}
                  ${header.className || ''}
                `}
              >
                {header.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.map((item, index) => renderRow(item, index))}
        </tbody>
      </table>
    </div>
  );
}

export default DataTable;
