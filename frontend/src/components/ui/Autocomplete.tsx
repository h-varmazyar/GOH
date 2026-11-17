import { useState, useRef, useEffect } from 'react';

interface AutocompleteProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit?: () => void;
  suggestions: string[];
  placeholder?: string;
  className?: string;
}

/**
 * Token-aware autocomplete: shows suggestions based on the last token in the input.
 * A token is split by whitespace, comparison operators, parentheses, etc.
 */
export function Autocomplete({
  value,
  onChange,
  onSubmit,
  suggestions,
  placeholder,
  className = '',
}: AutocompleteProps) {
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // Find the current token being typed (after last space, operator, or paren)
  const getCurrentToken = (text: string): { token: string; start: number } => {
    const match = text.match(/[A-Za-z_][\w.]*$/);
    if (match) {
      return { token: match[0], start: match.index || 0 };
    }
    return { token: '', start: text.length };
  };

  const { token } = getCurrentToken(value);

  const filteredSuggestions = token
    ? suggestions.filter((s) => s.toLowerCase().startsWith(token.toLowerCase()) && s !== token)
    : suggestions.slice(0, 10);

  // Close on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  useEffect(() => {
    setActiveIndex(0);
  }, [token, value]);

  const applySuggestion = (suggestion: string) => {
    const { start } = getCurrentToken(value);
    const before = value.slice(0, start);
    // Add a dot after indicator name to start property
    const completion = suggestion.endsWith('.') ? suggestion : suggestion;
    const newValue = before + completion;
    onChange(newValue);
    setOpen(false);
    inputRef.current?.focus();
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (open && filteredSuggestions.length > 0) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setActiveIndex((i) => (i + 1) % filteredSuggestions.length);
        return;
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        setActiveIndex((i) => (i - 1 + filteredSuggestions.length) % filteredSuggestions.length);
        return;
      }
      if (e.key === 'Tab' || (e.key === 'Enter' && !e.shiftKey)) {
        e.preventDefault();
        applySuggestion(filteredSuggestions[activeIndex]);
        return;
      }
      if (e.key === 'Escape') {
        setOpen(false);
        return;
      }
    }
    if (e.key === 'Enter' && !open && onSubmit) {
      e.preventDefault();
      onSubmit();
    }
  };

  return (
    <div ref={containerRef} className={`relative ${className}`}>
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={(e) => {
          onChange(e.target.value);
          setOpen(true);
        }}
        onFocus={() => setOpen(true)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        dir="ltr"
        role="combobox"
        aria-expanded={open && filteredSuggestions.length > 0}
        aria-autocomplete="list"
        className="field font-mono"
      />

      {open && filteredSuggestions.length > 0 && (
        <div role="listbox" className="absolute z-(--z-dropdown) mt-1 max-h-56 w-full overflow-y-auto rounded-md border border-border bg-surface py-1 shadow-md">
          {filteredSuggestions.map((s, i) => (
            <button
              key={s}
              type="button"
              role="option"
              aria-selected={i === activeIndex}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => applySuggestion(s)}
              className={`flex min-h-9 w-full items-center px-3 text-start font-mono text-body transition-colors
                ${i === activeIndex
                  ? 'bg-selected/15 text-text'
                  : 'text-text-muted hover:bg-surface-muted'
                }`}
              dir="ltr"
            >
              {s}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
