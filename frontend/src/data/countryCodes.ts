export interface CountryCode {
  code: string;    // ISO country code
  name: string;    // Persian name
  dialCode: string; // e.g. "+98"
  flag: string;    // Emoji flag
}

export const countryCodes: CountryCode[] = [
  { code: 'IR', name: 'ایران', dialCode: '+98', flag: '🇮🇷' },
  { code: 'TR', name: 'ترکیه', dialCode: '+90', flag: '🇹🇷' },
  { code: 'AE', name: 'امارات', dialCode: '+971', flag: '🇦🇪' },
  { code: 'SA', name: 'عربستان', dialCode: '+966', flag: '🇸🇦' },
  { code: 'IQ', name: 'عراق', dialCode: '+964', flag: '🇮🇶' },
  { code: 'AF', name: 'افغانستان', dialCode: '+93', flag: '🇦🇫' },
  { code: 'PK', name: 'پاکستان', dialCode: '+92', flag: '🇵🇰' },
  { code: 'IN', name: 'هند', dialCode: '+91', flag: '🇮🇳' },
  { code: 'US', name: 'آمریکا', dialCode: '+1', flag: '🇺🇸' },
  { code: 'GB', name: 'انگلستان', dialCode: '+44', flag: '🇬🇧' },
  { code: 'DE', name: 'آلمان', dialCode: '+49', flag: '🇩🇪' },
  { code: 'FR', name: 'فرانسه', dialCode: '+33', flag: '🇫🇷' },
  { code: 'IT', name: 'ایتالیا', dialCode: '+39', flag: '🇮🇹' },
  { code: 'ES', name: 'اسپانیا', dialCode: '+34', flag: '🇪🇸' },
  { code: 'NL', name: 'هلند', dialCode: '+31', flag: '🇳🇱' },
  { code: 'CA', name: 'کانادا', dialCode: '+1', flag: '🇨🇦' },
  { code: 'AU', name: 'استرالیا', dialCode: '+61', flag: '🇦🇺' },
  { code: 'RU', name: 'روسیه', dialCode: '+7', flag: '🇷🇺' },
  { code: 'CN', name: 'چین', dialCode: '+86', flag: '🇨🇳' },
  { code: 'JP', name: 'ژاپن', dialCode: '+81', flag: '🇯🇵' },
  { code: 'KR', name: 'کره جنوبی', dialCode: '+82', flag: '🇰🇷' },
  { code: 'MY', name: 'مالزی', dialCode: '+60', flag: '🇲🇾' },
  { code: 'GE', name: 'گرجستان', dialCode: '+995', flag: '🇬🇪' },
  { code: 'AM', name: 'ارمنستان', dialCode: '+374', flag: '🇦🇲' },
  { code: 'AZ', name: 'آذربایجان', dialCode: '+994', flag: '🇦🇿' },
];

export const defaultCountryCode = countryCodes[0]; // Iran
