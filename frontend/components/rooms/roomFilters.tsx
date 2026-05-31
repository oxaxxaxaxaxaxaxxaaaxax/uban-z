'use client';

import { Box, TextField, FormControl, InputLabel, Select, MenuItem, IconButton } from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import CloseIcon from '@mui/icons-material/Close';
import { useState, useEffect, ChangeEvent } from 'react';
import styles from './roomFilters.module.scss';

import type { RoomFilters } from '@/hooks/useFilteredRooms';

interface RoomFiltersProps {
  value: RoomFilters;
  onChange: (filters: RoomFilters) => void;
  availableBuildings?: string[];
}

export default function RoomFilters({value, onChange, availableBuildings = []}: RoomFiltersProps) {

  const [searchInput, setSearchInput] = useState(value.search);

  useEffect(() => {
    setSearchInput(value.search);
  }, [value.search]);

  const handleSearchChange = (e: ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    setSearchInput(newValue);
    onChange({ ...value, search: newValue });
  };

  const handleClearSearch = () => {
    setSearchInput('');
    onChange({ ...value, search: '' });
  };

  return (
    <Box className={styles.container}>

      <Box className={styles.searchWrapper}>
        <TextField
          variant="outlined"
          fullWidth
          size="small"
          placeholder="Поиск аудитории (название, номер)..."
          value={searchInput}
          onChange={handleSearchChange}
          className={styles.searchField}
          slotProps={{
            input: {
              startAdornment: (
                <SearchIcon className={styles.searchIcon} />
              ),
              endAdornment: searchInput ? (
                <IconButton size="small" onClick={handleClearSearch} edge="end">
                  <CloseIcon fontSize="small" />
                </IconButton>
              ) : null,
            },
          }}
        />
      </Box>

      <Box className={styles.controls}>

        {/* Корпус */}
        <FormControl size="small" className={styles.control}>
          <InputLabel>Корпус</InputLabel>
          <Select
            value={value.building || 'all'}
            label="Корпус"
            onChange={(e) => onChange({
              ...value,
              building: e.target.value === 'all' ? undefined : e.target.value
            })}
          >
            <MenuItem value="all">Все корпуса</MenuItem>
            {availableBuildings.map((b) => (
              <MenuItem key={b} value={b}>{b}</MenuItem>
            ))}
          </Select>
        </FormControl>

        {/* Мин. вместимость */}
        <FormControl size="small" className={styles.control}>
          <InputLabel>Мин. мест</InputLabel>
          <Select
            value={value.minCapacity || 'any'}
            label="Мин. мест"
            onChange={(e) => {
              const val = e.target.value;
              onChange({
                ...value,
                minCapacity: val === 'any' ? undefined : Number(val)
              });
            }}
          >
            <MenuItem value="any">Любая</MenuItem>
            <MenuItem value={20}>от 15</MenuItem>
            <MenuItem value={20}>от 30</MenuItem>
            <MenuItem value={50}>от 50</MenuItem>
            <MenuItem value={100}>от 100</MenuItem>
          </Select>
        </FormControl>

        {/* Кнопка сброса */}
        {(value.search || value.building || value.minCapacity) && (
          <IconButton
            size="small"
            onClick={() => onChange({
              search: '',
              building: undefined,
              minCapacity: undefined,
            })}
            className={styles.resetBtn}
            title="Сбросить фильтры"
          >
            <CloseIcon fontSize="small" />
          </IconButton>
        )}
      </Box>

    </Box>
  );
}
