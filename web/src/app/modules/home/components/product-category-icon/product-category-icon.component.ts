import { Component, Input, OnInit } from '@angular/core';

@Component({
  selector: 'app-product-category-icon',
  templateUrl: './product-category-icon.component.html',
  styleUrls: ['./product-category-icon.component.scss'],
})
export class ProductCategoryIconComponent implements OnInit {

  private _category: string;

  @Input() set category(value: string) {
    this._category = value;
  }

  get category(): string {
    return this._category;
  }

  constructor() { }

  ngOnInit() {
  }

}
