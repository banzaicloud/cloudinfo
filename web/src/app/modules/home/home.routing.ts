import { RouterModule, Routes } from '@angular/router';
import { ModuleWithProviders } from '@angular/core';

import { HomeComponent } from './containers/home/home.component';
import { ProductListComponent } from './components/product-list/product-list.component';

export const routes: Routes = [
  {
    path: '', component: HomeComponent, children: [
      { path: '', component: ProductListComponent },
    ],
  },
];

export const routing: ModuleWithProviders = RouterModule.forChild(routes);
