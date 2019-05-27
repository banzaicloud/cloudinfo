import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NavigationBarComponent } from './components/navigation-bar/navigation-bar.component';
import { PageTitleSectionComponent } from './components/page-title-section/page-title-section.component';
import { SearchModule } from '../search/search.module';

@NgModule({
  imports: [
    CommonModule,
    SearchModule,
  ],
  declarations: [
    NavigationBarComponent,
    PageTitleSectionComponent,
  ],
  exports: [
    NavigationBarComponent,
  ],
})
export class NavBarModule {
}
