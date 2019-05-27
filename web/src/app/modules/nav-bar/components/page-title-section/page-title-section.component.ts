import { Component, Input } from '@angular/core';

@Component({
  selector: 'app-page-title-section',
  templateUrl: './page-title-section.component.html',
  styleUrls: ['./page-title-section.component.scss'],
})
export class PageTitleSectionComponent {

  @Input() title: string;
  @Input() searchField: boolean;

  constructor() { }

}
