import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../environments/environment';
import { Region } from '../models/region';
import { Provider } from '../models/provider';
import { DisplayedProduct, Products } from '../models/product';

@Injectable({
  providedIn: 'root',
})
export class CloudInfoService {

  private readonly productsUrlBase: string;
  private scrapingTime$ = new BehaviorSubject<number>(null);

  constructor(private http: HttpClient) {
    this.productsUrlBase = environment.baseProductUrl;
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

  public getProducts(provider: string, service: string, region: string): Observable<DisplayedProduct[]> {
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
            const displayedSpot = avgSpot && avgSpot !== 0 ? avgSpot : 'unavailable';

            return {
              category: response.category,
              type: response.type,
              cpu: response.cpusPerVm,
              mem: response.memPerVm,
              regularPrice: response.onDemandPrice,
              spotPrice: displayedSpot,
              ntwPerf: response.ntwPerf === '' ? 'unavailable' : response.ntwPerf,
            };
          });
      }),
    );
  }
}
