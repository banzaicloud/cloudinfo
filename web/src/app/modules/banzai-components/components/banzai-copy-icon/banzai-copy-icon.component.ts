import { Component, Input, OnInit } from '@angular/core';

@Component({
  selector: 'app-banzai-copy-icon',
  templateUrl: './banzai-copy-icon.component.html',
  styleUrls: ['./banzai-copy-icon.component.scss'],
})
export class BanzaiCopyIconComponent implements OnInit {

  @Input() value: string;
  @Input() disabled: boolean;
  @Input() size = 24;
  @Input() paddingBottom = 0;

  constructor() { }

  ngOnInit() {
  }

}
