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

import {Injectable} from '@angular/core';
import {DisplayedProduct, Products, Region, Provider} from './product';
import {Observable, BehaviorSubject} from 'rxjs';
import {map} from 'rxjs/operators';
import {HttpClient} from '@angular/common/http';

@Injectable({
  providedIn: 'root'
})
export class ProductService {

  private productsUrlBase = 'api/v1/';
  private scrapingTime$ = new BehaviorSubject<number>(null);

  constructor(private http: HttpClient) {
  }

  public getScrapingTime() {
    return this.scrapingTime$.asObservable();
  }

  public getRegions(provider: string, service: string): Observable<Region[]> {
    return this.http.get<Region[]>(this.productsUrlBase + `providers/${provider}/services/${service}/regions`);
  }

  public getProviders(): Observable<{ providers: Provider[] }> {
    return this.http.get<{ providers: Provider[] }>(this.productsUrlBase + 'providers');
  }

  getProducts(provider: string, service: string, region: string): Observable<DisplayedProduct[]> {
    return this.http.get<Products>(this.productsUrlBase + `providers/${provider}/services/${service}/regions/${region}/products`).pipe(
      map(res => {
        if (res.scrapingTime) {
          this.scrapingTime$.next(+res.scrapingTime);
        }
        return res.products.map(
          response => {
            let avgSpot = 0;
            if (response.spotPrice != null) {
              let i;
              for (i = 0; i < response.spotPrice.length; i++) {
                avgSpot = avgSpot + response.spotPrice[i].price;
              }
              avgSpot = avgSpot / response.spotPrice.length;
            }
            const displayedSpot = avgSpot !== 0 ? avgSpot : 'unavailable';

            return new DisplayedProduct(
              response.type,
              response.cpusPerVm,
              response.cpusPerVm + ' vCPUs',
              response.memPerVm,
              response.memPerVm + ' GB',
              response.onDemandPrice,
              displayedSpot,
              response.ntwPerf === '' ? 'unavailable' : response.ntwPerf);
          });
      })
    );
  }
}
