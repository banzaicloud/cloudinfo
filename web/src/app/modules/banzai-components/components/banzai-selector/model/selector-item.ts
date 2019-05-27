export interface SelectorItem {
  label: string; // label in select option(s)
  display: string; // select title after selection
  value: string; // select option content
  id: number; // this is necessary, multiple pke values
}

export interface SelectorGroup {
  label?: string;
  value?: string;
  items: SelectorItem[];
}
