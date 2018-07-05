import {NgModule} from '@angular/core';
import {BrowserModule} from '@angular/platform-browser';
import {BrowserAnimationsModule} from '@angular/platform-browser/animations';
import {MatToolbarModule} from '@angular/material/toolbar';
import {MatTableModule} from '@angular/material/table';
import {MatSelectModule} from '@angular/material/select';
import {MatExpansionModule} from '@angular/material/expansion';
import {MatInputModule} from '@angular/material/input';
import {MatIconModule} from '@angular/material/icon';
import {MatSortModule} from '@angular/material/sort';
import {HttpClientModule} from '@angular/common/http';

import {AppComponent} from './app.component';
import {ProductsComponent} from './products/products.component';
import { ToFixedNumberPipe } from './products/toFixedNumber.pipe';

@NgModule({
  declarations: [
    AppComponent,
    ProductsComponent,
    ToFixedNumberPipe,
  ],
  imports: [
    BrowserModule,
    BrowserAnimationsModule, // new modules added here
    MatToolbarModule,
    MatTableModule,
    MatSelectModule,
    MatInputModule,
    MatExpansionModule,
    MatSortModule,
    MatIconModule,
    HttpClientModule
  ],
  providers: [],
  bootstrap: [AppComponent]
})
export class AppModule {
}
