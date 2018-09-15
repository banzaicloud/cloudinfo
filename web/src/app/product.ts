/*
 * Copyright © 2018 Banzai Cloud
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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

export interface Provider {
  provider: string;
  name?: string;
  services: Array<{ service: string }>;
}


