import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HomeComponent } from './containers/home/home.component';
import { routing } from './home.routing';
import { ProductListComponent } from './components/product-list/product-list.component';
import { NavBarModule } from '../nav-bar/nav-bar.module';
import { BanzaiComponentsModule } from '../banzai-components/banzai-components.module';
import { ProductCategoryIconComponent } from './components/product-category-icon/product-category-icon.component';
import { CategoryIconPipe } from './components/product-category-icon/pipe/category-icon.pipe';
import { MatTooltipModule } from '@angular/material';

@NgModule({
  imports: [
    CommonModule,
    NavBarModule,
    BanzaiComponentsModule,
    MatTooltipModule,
    routing,
  ],
  declarations: [
    HomeComponent,
    ProductListComponent,
    ProductCategoryIconComponent,
    CategoryIconPipe,
  ],
  providers: [
    CategoryIconPipe,
  ],
})
export class HomeModule {
}
