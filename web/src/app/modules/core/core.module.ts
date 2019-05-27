import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NotFoundComponent } from './components/not-found/not-found.component';
import { NavBarModule } from '../nav-bar/nav-bar.module';

@NgModule({
  imports: [
    CommonModule,
    NavBarModule,
  ],
  declarations: [
    NotFoundComponent,
  ],
  exports: [
    NotFoundComponent,
  ],
})
export class CoreModule {
}
