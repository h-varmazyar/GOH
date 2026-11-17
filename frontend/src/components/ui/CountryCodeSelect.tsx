import { useEffect, useId, useRef, useState } from 'react';
import { ChevronDown } from 'lucide-react';
import { countryCodes, type CountryCode } from '../../data/countryCodes';

interface CountryCodeSelectProps {
  value: CountryCode;
  onChange: (country: CountryCode) => void;
  /** Label text for screen readers; the visible label is rendered by the parent. */
  ariaLabel?: string;
}

/**
 * Dial-code combobox: shows the selected flag + dial code, and lets the user
 * type a dial code, country name or ISO code to filter the dropdown.
 */
export function CountryCodeSelect({ value, onChange, ariaLabel = 'کد کشور' }: CountryCodeSelectProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [activeIndex, setActiveIndex] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const listId = useId();
  // Fixed positioning keeps the list from being clipped by a scrolling parent (e.g. a modal body)
  const [menuStyle, setMenuStyle] = useState<React.CSSProperties>({});

  const query = search.trim().toLowerCase();
  const digits = query.replace(/^\+/, '');
  const filtered = query
    ? countryCodes.filter(
        (c) =>
          c.dialCode.replace('+', '').startsWith(digits) ||
          c.name.includes(search.trim()) ||
          c.code.toLowerCase().includes(query)
      )
    : countryCodes;

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) close();
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  // A fixed menu would drift from the field on scroll or resize, so close it instead
  useEffect(() => {
    if (!open) return;
    const onMove = (e: Event) => {
      if (e.target instanceof Node && containerRef.current?.contains(e.target)) return;
      close();
    };
    window.addEventListener('resize', onMove);
    window.addEventListener('scroll', onMove, true);
    return () => {
      window.removeEventListener('resize', onMove);
      window.removeEventListener('scroll', onMove, true);
    };
  }, [open]);

  function openMenu() {
    const rect = containerRef.current?.getBoundingClientRect();
    if (rect) {
      const menuWidth = 256;
      const menuMaxHeight = 240;
      const left = Math.max(8, Math.min(rect.left, window.innerWidth - menuWidth - 8));
      const fitsBelow = window.innerHeight - rect.bottom >= menuMaxHeight + 8;
      setMenuStyle(
        fitsBelow
          ? { left, top: rect.bottom + 4 }
          : { left, bottom: window.innerHeight - rect.top + 4 }
      );
    }
    setOpen(true);
  }

  function close() {
    setOpen(false);
    setSearch('');
    setActiveIndex(0);
  }

  function select(country: CountryCode) {
    onChange(country);
    close();
    inputRef.current?.blur();
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (!open) openMenu();
      setActiveIndex((i) => Math.min(i + 1, filtered.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setActiveIndex((i) => Math.max(i - 1, 0));
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (open && filtered[activeIndex]) select(filtered[activeIndex]);
    } else if (e.key === 'Escape') {
      // Close only the list, not an enclosing dialog
      if (open) e.stopPropagation();
      close();
    } else if (e.key === 'Tab') {
      close();
    }
  };

  return (
    <div ref={containerRef} className="relative w-full" dir="ltr">
      <span
        className="pointer-events-none absolute inset-y-0 start-3 flex items-center text-h3 leading-none"
        aria-hidden="true"
      >
        {value.flag}
      </span>
      <input
        ref={inputRef}
        type="text"
        role="combobox"
        aria-label={ariaLabel}
        aria-expanded={open}
        aria-controls={listId}
        autoComplete="off"
        value={open ? search : value.dialCode}
        placeholder={value.dialCode}
        onFocus={openMenu}
        onClick={() => !open && openMenu()}
        onChange={(e) => {
          setSearch(e.target.value);
          setActiveIndex(0);
          if (!open) openMenu();
        }}
        onKeyDown={handleKeyDown}
        className="field ps-11 pe-8 tabular-nums"
      />
      <ChevronDown
        className={`pointer-events-none absolute end-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-text-subtle transition-transform ${open ? 'rotate-180' : ''}`}
        aria-hidden="true"
      />

      {open && (
        <ul
          id={listId}
          role="listbox"
          style={menuStyle}
          className="fixed z-(--z-dropdown) max-h-60 w-64 overflow-y-auto rounded-md border border-border-strong bg-surface py-1 shadow-lg"
        >
          {filtered.length === 0 ? (
            <li className="px-3 py-2 text-body text-text-subtle" dir="rtl">
              کشوری یافت نشد
            </li>
          ) : (
            filtered.map((country, index) => {
              const isSelected = country.code === value.code;
              return (
                <li
                  key={country.code}
                  role="option"
                  aria-selected={isSelected}
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => select(country)}
                  onMouseEnter={() => setActiveIndex(index)}
                  className={`flex cursor-pointer items-center gap-2 px-3 py-2 text-body transition-colors
                    ${index === activeIndex ? 'bg-surface-muted' : ''}
                    ${isSelected ? 'bg-selected/10' : ''}`}
                >
                  <span className="text-h3 leading-none">{country.flag}</span>
                  <span className="w-12 tabular-nums text-text-subtle">{country.dialCode}</span>
                  <span className="ms-auto text-text" dir="rtl">
                    {country.name}
                  </span>
                </li>
              );
            })
          )}
        </ul>
      )}
    </div>
  );
}
