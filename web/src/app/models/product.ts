export class Products {
  products: Product[];
  scrapingTime: string;
}

export class Product {
  category: string;
  type: string;
  cpusPerVm: number;
  memPerVm: number;
  onDemandPrice: number;
  spotPrice: SpotPrice[];
  ntwPerf: string;
}

export class SpotPrice {
  zone: string;
  price: number;
}

export interface DisplayedProduct {
  category: string;
  type: string;
  cpu: number;
  mem: number;
  regularPrice: number;
  spotPrice: number | string;
  ntwPerf: string;
}
