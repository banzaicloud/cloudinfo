import {Component, OnInit, ViewChild} from '@angular/core';
import {ProductService} from '../product.service';
import {DisplayedProduct, Region} from '../product';
import {Observable} from 'rxjs/index';
import {MatSort, MatSortable, MatTableDataSource} from '@angular/material';

@Component({
  selector: 'app-products',
  templateUrl: './products.component.html',
  styleUrls: ['./products.component.scss'],
})
export class ProductsComponent implements OnInit {

  columnsToDisplay = ['type', 'cpu', 'mem', 'ntwPerf', 'regularPrice', 'spotPrice'];

  regions: Region[];
  provider: string = 'ec2';
  region: string;
  products: MatTableDataSource<DisplayedProduct>;

  constructor(private productService: ProductService) {
  }

  @ViewChild(MatSort) sort: MatSort;

  ngOnInit() {
    this.updateProducts();
  }

  getRegions(): Observable<Region[]> {
    return new Observable(observer => {
      this.productService.getRegions(this.provider)
        .subscribe(regions => {
          this.regions = this.sortRegions(regions);
          this.region = regions[0].id;
          observer.next(regions);
        });
    });
  }

  getProducts(): void {
    this.productService.getProducts(this.provider, this.region)
      .subscribe(products => {
        this.products = new MatTableDataSource<DisplayedProduct>(products);
        this.products.sort = this.sort;
      });
  }

  updateProducts(): void {
    this.getRegions().subscribe(() => {
      this.getProducts();
    });
  }

  applyFilter(filterValue: string) {
    filterValue = filterValue.trim();
    filterValue = filterValue.toLowerCase();
    this.products.filter = filterValue;
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
}
