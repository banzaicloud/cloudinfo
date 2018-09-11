import { Component, OnInit, ViewChild } from '@angular/core';
import { ProductService } from '../product.service';
import { DisplayedProduct, Region, Provider } from '../product';
import { Observable } from 'rxjs/index';
import { MatSort, MatTableDataSource, MatSelectChange } from '@angular/material';
import { switchMap } from 'rxjs/operators';
import { PROVIDERS } from '../constants/providers';

@Component({
  selector: 'app-products',
  templateUrl: './products.component.html',
  styleUrls: ['./products.component.scss'],
})
export class ProductsComponent implements OnInit {

  columnsToDisplay = ['type', 'cpu', 'mem', 'ntwPerf', 'regularPrice', 'spotPrice'];

  regions: Region[];
  providers: Provider[] = [];
  selectedProvider = '';
  selectedService = '';
  selectedRegion = '';
  products: MatTableDataSource<DisplayedProduct>;
  scrapingTime: Observable<number>;

  constructor(private productService: ProductService) {
  }

  @ViewChild(MatSort) sort: MatSort;

  ngOnInit() {
    this.initializeData();
    this.scrapingTime = this.productService.getScrapingTime();
  }

  private initializeData() {
    this.productService.getProviders()
      .pipe(
        switchMap(response => {
          const providersList = response.providers;
          this.providers = this.mapProviderList(providersList);
          this.selectedProvider = providersList[0].provider;
          this.selectedService = providersList[0].services[0].service;
          return this.getRegions();
        })
      )
      .subscribe(regions => {
        this.regions = this.sortRegions(regions);
        this.selectedRegion = regions[0].id;
        this.getProducts();
      },
      error => {
        console.log(`Error during getting providers/regions`, error);
      });
  }

  public getRegions(): Observable<Region[]> {
    return this.productService.getRegions(this.selectedProvider, this.selectedService);
  }

  public getProducts(): void {
    this.productService.getProducts(this.selectedProvider, this.selectedService, this.selectedRegion)
      .subscribe(products => {
        this.products = new MatTableDataSource<DisplayedProduct>(products);
        this.products.sort = this.sort;
      });
  }

  public updateProducts(service: string, provider: string): void {
    this.selectedService = service;
    this.selectedProvider = provider;
    this.getRegions().subscribe(regions => {
      this.regions = this.sortRegions(regions);
      this.selectedRegion = regions[0].id;
      this.getProducts();
    });
  }

  public applyFilter(filterValue: string) {
    filterValue = filterValue.trim();
    filterValue = filterValue.toLowerCase();
    this.products.filter = filterValue;
  }

  private mapProviderList(providers: Provider[]): Provider[] {
    return providers.map(provider => {
      return this.addProviderDisplayName(provider);
    });
  }

  private sortRegions(regions: Region[]): Region[] {
    return regions.sort((a, b) => {
      const nameA = a.name.toUpperCase();
      const nameB = b.name.toUpperCase();
      if (nameA < nameB) {
        return -1;
      }
      if (nameA > nameB) {
        return 1;
      }
      return 0;
    });
  }

  private addProviderDisplayName(provider: Provider): Provider {
    switch (provider.provider) {
      case PROVIDERS.amazon.provider: {
        provider.name = PROVIDERS.amazon.name;
        break;
      }

      case PROVIDERS.alibaba.provider: {
        provider.name = PROVIDERS.alibaba.name;
        break;
      }

      case PROVIDERS.google.provider: {
        provider.name = PROVIDERS.google.name;
        break;
      }

      case PROVIDERS.azure.provider: {
        provider.name = PROVIDERS.azure.name;
        break;
      }

      case PROVIDERS.oracle.provider: {
        provider.name = PROVIDERS.oracle.name;
        break;
      }

      default: {
        provider.name = provider.provider;
        break;
      }
    }
    return provider;
  }
}
