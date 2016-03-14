// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import {
  stateName as daemonsets,
} from 'daemonsetlist/daemonsetlist_state';

/**
 * Controller for the daemonset info directive.
 * @final
 */
export default class DaemonSetInfoController {
  /**
   * @param {!./daemonsetdetail_state.StateParams} $stateParams
   * @param {!ui.router.$state} $state
   * @param {!angular.$log} $log
   * @param {!./daemonset_service.DaemonSetService}
   * kdDaemonSetService
   *
   * @ngInject
   */
  constructor($stateParams, $state, $log, kdDaemonSetService) {
    /** @private {!./daemonsetdetail_state.StateParams} */
    this.stateParams_ = $stateParams;

    /** @private {!./daemonset_service.DaemonSetService} */
    this.kdDaemonSetService_ = kdDaemonSetService;

    /** @private {!ui.router.$state} */
    this.state_ = $state;

    /** @private {!angular.$log} */
    this.log_ = $log;

    /**
     * Daemon set details. Initialized from the scope.
     * @export {!backendApi.DaemonSetDetail}
     */
    this.details;
  }

  /**
   * @return {boolean}
   * @export
   */
  areDesiredPodsRunning() { return this.details.podInfo.running === this.details.podInfo.desired; }

  /**
   * Callbacks used after clicking dialog confirmation button in order to delete daemonset
   * or log unsuccessful operation error.
   */

  /**
   * Changes state back to daemonset list after successful deletion of daemonset.
   * @private
   */
  onDaemonSetDeleteSuccess_() {
    this.log_.info('Daemon set successfully deleted.');
    this.state_.go(daemonsets);
  }
}
