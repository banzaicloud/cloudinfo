import { ChangeDetectorRef, Component, ElementRef, NgZone, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { Subject } from 'rxjs';
import { SelectorGroup, SelectorItem } from '../../../banzai-components/components/banzai-selector/model/selector-item';
import { distinctUntilChanged, finalize, pluck, takeUntil } from 'rxjs/operators';
import { CloudInfoService } from '../../../../services/cloud-info.service';
import { Provider } from '../../../../models/provider';
import { Region } from '../../../../models/region';
import { PROVIDERS } from '../../../../constants/providers';
import { TableData } from '../../../banzai-components/components/banzai-table/model/tabledata';
import { ProductsListFactory } from '../../../banzai-components/components/banzai-table/factories/products-list-factory';
import { DisplayedProduct } from '../../../../models/product';
import { TimeAgoPipe } from 'time-ago-pipe';
import { DatePipe } from '@angular/common';
import { ActivatedRoute } from '@angular/router';

@Component({
  selector: 'app-product-list',
  templateUrl: './product-list.component.html',
  styleUrls: ['./product-list.component.scss'],
})
export class ProductListComponent implements OnInit, OnDestroy {

  @ViewChild('categoryTemplate') categoryTemplate: ElementRef;

  private readonly currentURL: string;
  private unsubscribe$ = new Subject();
  private timeAgoPipe: TimeAgoPipe;
  private selectedProvider: string;
  private selectedService: string;
  private selectedRegion: string;

  public isProviderLoading: boolean;
  public isProductsLoading: boolean;
  public isRegionLoading: boolean;
  public products: DisplayedProduct[];
  public productsTableData: TableData;
  public providers: SelectorGroup[];
  public regions: SelectorGroup[];
  public scrapingTimeTooltip: string;
  public scrapingTime: string;
  public searchValue: string;
  public cUrlCommand: string;

  constructor(
    private changeDetectorRef: ChangeDetectorRef,
    private cloudInfoService: CloudInfoService,
    private activatedRoute: ActivatedRoute,
    private datePipe: DatePipe,
    private zone: NgZone,
  ) {
    this.timeAgoPipe = new TimeAgoPipe(changeDetectorRef, zone);
    this.currentURL = window.location.href.split('?')[0].replace(/\/$/, '');
  }

  ngOnInit() {
    this.loadProviders();
    this.listenToScrapingTime();
    this.listenOnFilterChanges();
  }

  private listenToScrapingTime() {
    this.cloudInfoService.getScrapingTime()
      .pipe(takeUntil(this.unsubscribe$))
      .subscribe(
        time => {
          if (time && time !== 0) {
            const date = this.datePipe.transform(time, 'medium');
            this.scrapingTimeTooltip = date;
            this.scrapingTime = this.timeAgoPipe.transform(date);
          } else {
            this.scrapingTime = '';
            this.scrapingTimeTooltip = '';
          }
        },
        err => console.error('error during loading scraping time: ', err),
      );
  }

  private loadProviders() {
    this.isProductsLoading = true;
    this.isProviderLoading = true;
    this.isRegionLoading = true;
    this.cloudInfoService.getProviders()
      .pipe(
        takeUntil(this.unsubscribe$),
        finalize(() => this.isProviderLoading = false),
      )
      .subscribe(
        (res: { providers: Provider[] }) => {
          this.providers = this.mapProvidersToSelector(res.providers);
        },
        (err) => {
          console.error('error during loading providers: ', err);
          this.isRegionLoading = false;
          this.isProductsLoading = false;
        },
      );
  }

  private mapProvidersToSelector(providers: Provider[]): SelectorGroup[] {
    let index = 0;
    return providers.map(p => {
      const providerName = this.getProviderDisplayName(p.provider);
      const afterMap = this.convertServicesToSelectors(providerName, p.services, index);
      index = afterMap.index;
      return {
        label: providerName,
        value: p.provider,
        items: afterMap.items,
      };
    });
  }

  private convertServicesToSelectors(
    provider: string,
    values: { service: string }[],
    indexStart: number,
  ): { items: SelectorItem[], index: number } {
    let index = indexStart;
    const items: SelectorItem[] = values.map(v => {
      return {
        label: v.service,
        display: `${provider} - ${v.service}`,
        value: v.service,
        id: index += 1,
      };
    });

    return { items, index };
  }

  private getProviderDisplayName(provider: string): string {
    switch (provider) {
      case PROVIDERS.amazon.provider: {
        return PROVIDERS.amazon.name;
      }

      case PROVIDERS.alibaba.provider: {
        return PROVIDERS.alibaba.name;
      }

      case PROVIDERS.google.provider: {
        return PROVIDERS.google.name;
      }

      case PROVIDERS.azure.provider: {
        return PROVIDERS.azure.name;
      }

      case PROVIDERS.oracle.provider: {
        return PROVIDERS.oracle.name;
      }

      case PROVIDERS.digitalocean.provider: {
        return PROVIDERS.digitalocean.name;
      }

      default: {
        return provider;
      }
    }
  }

  public providerChanged(event: { group: string, item: SelectorItem }) {

    this.selectedProvider = event.group;
    this.selectedService = event.item.value;

    this.loadRegions(event.group, event.item.value);

  }

  private loadRegions(provider: string, service: string) {
    this.isProductsLoading = true;
    this.isRegionLoading = true;
    this.cloudInfoService.getRegions(provider, service)
      .pipe(
        takeUntil(this.unsubscribe$),
        finalize(() => this.isRegionLoading = false),
      )
      .subscribe(
        (regions: Region[]) => {
          this.regions = this.mapRegionsToSelector(this.sortRegions(regions));
        },
        (err) => {
          console.error('error during loading regions: ', err);
          this.isProductsLoading = false;
        },
      );
  }

  public regionChanged(event: { group: string, item: SelectorItem }) {
    this.selectedRegion = event.item.value;
    // tslint:disable-next-line:max-line-length
    this.cUrlCommand = `curl -L -X GET \'${this.currentURL}/api/v1/providers/${this.selectedProvider}/services/${this.selectedService}/regions/${this.selectedRegion}/products\'`;
    this.loadProducts();
  }

  private loadProducts() {
    this.isProductsLoading = true;
    this.cloudInfoService.getProducts(this.selectedProvider, this.selectedService, this.selectedRegion)
      .pipe(
        takeUntil(this.unsubscribe$),
        finalize(() => this.isProductsLoading = false),
      )
      .subscribe(
        (products: DisplayedProduct[]) => {
          this.products = products;
          this.productsTableData = ProductsListFactory.generateTableConfig(products, this.categoryTemplate, this.selectedProvider);
        },
        err => console.error('error during loading products: ', err),
      );
  }

  private mapRegionsToSelector(regions: Region[]): SelectorGroup[] {
    return [
      {
        items: regions.map((r, index) => {
          return {
            label: r.name,
            display: r.name,
            value: r.id,
            id: index,
          };
        }),
      },
    ];
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

  private listenOnFilterChanges() {
    this.activatedRoute.queryParams
      .pipe(
        pluck('filter'),
        distinctUntilChanged(),
        takeUntil(this.unsubscribe$))
      .subscribe((filter: string = '') => {
        this.searchValue = filter;
      });
  }

  ngOnDestroy() {
    this.unsubscribe$.next();
    this.unsubscribe$.complete();
  }

}
