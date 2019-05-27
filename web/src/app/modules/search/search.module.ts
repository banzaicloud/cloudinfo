import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { SearchComponent } from './components/search/search.component';
import { FormsModule } from '@angular/forms';
import { MatIconModule } from '@angular/material';

@NgModule({
  imports: [
    CommonModule,
    MatIconModule,
    FormsModule,
  ],
  declarations: [
    SearchComponent,
  ],
  exports: [
    SearchComponent,
  ],
})
export class SearchModule {
}
