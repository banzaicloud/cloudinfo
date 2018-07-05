import {Injectable} from '@angular/core';
import {DisplayedProduct, Products, Region} from './product';
import {Observable} from 'rxjs';
import {map} from 'rxjs/operators';
import {HttpClient} from '@angular/common/http';

@Injectable({
  providedIn: 'root'
})
export class ProductService {

  private productsUrlBase = 'api/v1/';

  constructor(private http: HttpClient) {
  }

  getRegions(provider): Observable<Region[]> {
    return this.http.get<Region[]>(this.productsUrlBase + 'regions/' + provider);
  }

  getProducts(provider, region): Observable<DisplayedProduct[]> {
    return this.http.get<Products>(this.productsUrlBase + 'products/' + provider + '/' + region).pipe(
      map(res => {
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
