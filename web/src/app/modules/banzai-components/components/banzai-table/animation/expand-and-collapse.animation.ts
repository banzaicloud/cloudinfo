import { animate, state, style, transition, trigger } from '@angular/animations';

export const expandAndCollapseAnimation =
  trigger('expandAndCollapseAnimation', [
    state('collapsed, void', style({ height: '0px', minHeight: '0', opacity: 0, display: 'none' })),
    state('expanded', style({ height: '*', opacity: 1 })),
    transition('expanded <=> collapsed', animate('225ms cubic-bezier(0.4, 0.0, 0.2, 1)')),
    transition('expanded <=> void', animate('225ms cubic-bezier(0.4, 0.0, 0.2, 1)')),
  ]);
