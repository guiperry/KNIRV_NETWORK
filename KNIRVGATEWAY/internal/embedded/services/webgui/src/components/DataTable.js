import React from 'react';
import styles from './DataTable.module.css';

const DataTable = ({ 
  headers, 
  data, 
  renderRow 
}) => {
  return (
    <div className={styles.tableWrapper}>
      <table className={styles.dataTable}>
        <thead>
          <tr className={styles.tableRow}>
            {headers.map((header, index) => (
              <th 
                key={index} 
                className={`${styles.tableHeader} ${header.align === 'right' ? styles.textRight : header.align === 'center' ? styles.textCenter : ''}`}
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
};

export default DataTable;