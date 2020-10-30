import { Ref, computed } from "@vue/reactivity";
import { Store } from "../../../../core";
import { useWallet } from "../../hooks/useWallet";

export function useTokenListing({
  searchText,
  store,
  walletLimit,
  tokenLimit,
  selectedTokens,
}: {
  searchText: Ref<string>;
  store: Store;
  walletLimit: number;
  tokenLimit: number;
  selectedTokens: string[];
}) {
  const { balances } = useWallet(store);

  const walletTokens = computed(() => balances.value.map((tok) => tok.asset));
  const topTokens = computed(() => store.asset.topTokens);
  const fullTokenList = computed(() => {
    return Array.from(new Set([...walletTokens.value, ...topTokens.value]));
  });

  const limitedTokenList = computed(() => {
    return Array.from(
      new Set([
        ...walletTokens.value.slice(0, walletLimit),
        ...topTokens.value.slice(0, tokenLimit),
      ])
    );
  });

  const filteredTokens = computed(() => {
    const list = searchText.value
      ? fullTokenList.value.filter(
          ({ symbol }) =>
            symbol
              .toLowerCase()
              .indexOf(searchText.value.toLowerCase().trim()) > -1
        )
      : limitedTokenList.value;

    return list.map((item) =>
      selectedTokens.includes(item.symbol) ? { disabled: true, ...item } : item
    );
  });

  return { filteredTokens };
}