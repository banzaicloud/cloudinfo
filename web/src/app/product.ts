export class Products {
  products: Product[];
  scrapingTime: string;
}

export class Product {
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

export class DisplayedProduct {
  constructor(private type: string,
              private cpu: number,
              private cpuText: string,
              private mem: number,
              private memText: string,
              private regularPrice: number,
              private spotPrice: number | string,
              private ntwPerf: string) {
  }
}

export class Region {
  id: string;
  name: string;
}


