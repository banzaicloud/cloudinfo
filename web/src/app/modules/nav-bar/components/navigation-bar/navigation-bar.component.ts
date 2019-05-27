import { Component, ElementRef, Input, OnInit } from '@angular/core';

@Component({
  selector: 'app-navigation-bar',
  templateUrl: './navigation-bar.component.html',
  styleUrls: ['./navigation-bar.component.scss'],
})
export class NavigationBarComponent implements OnInit {

  @Input() searchField: boolean;
  @Input() title: string;

  constructor(
    private elementRef: ElementRef,
  ) { }

  ngOnInit() {
    this.addScript('https://buttons.github.io/buttons.js');
    this.addScript('https://platform.twitter.com/widgets.js');
  }

  private addScript(url: string) {
    const s = document.createElement('script');
    s.type = 'text/javascript';
    s.src = url;
    this.elementRef.nativeElement.appendChild(s);
  }

}
