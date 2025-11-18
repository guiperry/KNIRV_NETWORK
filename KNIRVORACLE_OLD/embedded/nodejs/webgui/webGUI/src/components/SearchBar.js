import React, { useState, useEffect, useRef } from 'react';
import styles from './SearchBar.module.css';

const SearchBar = ({ onSearch, placeholder = "Search..." }) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [isSearchActive, setIsSearchActive] = useState(false);
  const searchInputRef = useRef(null);

  useEffect(() => {
    if (isSearchActive && searchInputRef.current) {
      setTimeout(() => {
        searchInputRef.current.focus();
      }, 0);
    }
  }, [isSearchActive]);

  const handleSearchIconClick = () => {
    setIsSearchActive(!isSearchActive);
    setSearchQuery('');
    onSearch('');
  };

  const handleSearchChange = (event) => {
    const query = event.target.value;
    setSearchQuery(query);
    onSearch(query);
  };

  const handleClearSearch = () => {
    setSearchQuery('');
    onSearch('');
    if (searchInputRef.current) {
      searchInputRef.current.focus();
    }
  };

  return (
    <div className={styles.searchContainer}>
      {isSearchActive && (
        <input
          type="text"
          placeholder={placeholder}
          value={searchQuery}
          onChange={handleSearchChange}
          className={styles.searchInput}
          ref={searchInputRef}
        />
      )}
      <span className={styles.controlIcon} onClick={handleSearchIconClick}>
        {isSearchActive ? '×' : '🔍'}
      </span>
      {isSearchActive && searchQuery && (
        <span className={styles.clearSearchIcon} onClick={handleClearSearch}>
          &#10006;
        </span>
      )}
    </div>
  );
};

export default SearchBar;
