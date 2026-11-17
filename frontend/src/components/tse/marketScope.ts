import type { IndicatorScope, MarketRef } from '../../services/indicatorService';

export interface MarketScopeValue {
  scope: IndicatorScope;
  /** The selected markets; ignored for ALL, one market for MARKET. */
  markets: MarketRef[];
}

/** Checks a value; the message is shown under the market table. */
export const marketScopeError = (value: MarketScopeValue) => {
  if (value.scope === 'MARKET' && value.markets.length !== 1) return 'یک نماد انتخاب کنید.';
  if (value.scope === 'GROUP' && value.markets.length === 0) return 'نمادهای گروه را انتخاب کنید.';
  return undefined;
};

/** The market IDs to send for a value. */
export const marketIdsOf = (value: MarketScopeValue) => (value.scope === 'ALL' ? [] : value.markets.map((market) => market.id));
