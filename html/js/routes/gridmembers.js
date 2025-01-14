// Copyright 2019 Jason Ertel (github.com/jertel).
// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

const GridMemberAccepted = "accepted";
const GridMemberUnaccepted = "unaccepted";
const GridMemberRejected = "rejected";
const GridMemberDenied = "denied";

routes.push({ path: '/gridmembers', name: 'gridmembers', component: {
  template: '#page-gridmembers',
  data() { return {
    i18n: this.$root.i18n,
    members: [],
    denied: [],
    rejected: [],
    unaccepted: [],
    accepted: [],
    selected: null,
    dialog: false,
    confirmDeleteDialog: false,
    rules: {
      required: value => !!value || this.$root.i18n.required,
    },
  }},
  created() { 
    Vue.filter('colorNodeStatus', this.colorNodeStatus);
  },
  mounted() {
    this.$root.loadParameters("gridmembers", this.initGrid);
  },
  watch: {
    '$route': 'loadData',
  },
  methods: {
    initGrid(params) {
      this.loadData();
    },
    async loadData() {
      this.$root.startLoading();
      var route = this;
      try {
        const response = await this.$root.papi.get('gridmembers/');
        this.members = [];
        if (response && response.data && response.data.length > 0) {
          this.members = response.data;
          compFn = (a, b) => { return a.id != null ? a.id.localeCompare(b.id) : 0 };
          this.accepted = this.members.filter(node => node.status == GridMemberAccepted).sort(compFn);
          this.unaccepted = this.members.filter(node => node.status == GridMemberUnaccepted).sort(compFn);
          this.rejected = this.members.filter(node => node.status == GridMemberRejected).sort(compFn);
          this.denied = this.members.filter(node => node.status == GridMemberDenied).sort(compFn);
        }
      } catch (error) {
        this.$root.showError(error);
      }
      this.$root.stopLoading();
    },
    show(node) {
      this.selected = node;
      this.dialog = true;
    },
    hide() {
      this.dialog = false;
    },
    isUnaccepted(node) {
      return node.status == GridMemberUnaccepted;
    },
    confirmRemove() {
      this.confirmDeleteDialog = true;
    },
    cancelRemove() {
      this.confirmDeleteDialog = false;
    },
    async remove(node) {
      this.hide();
      this.cancelRemove();
      this.$root.startLoading();
      try {
        await this.$root.papi.post('gridmembers/' + node.id + "/delete");
        await this.loadData();
      } catch (error) {
        this.$root.showError(error);
      }
      this.$root.stopLoading();
    },
    async reject(node) {
      this.hide();
      this.$root.startLoading();
      try {
        await this.$root.papi.post('gridmembers/' + node.id + "/reject");
        await this.loadData();
      } catch (error) {
        this.$root.showError(error);
      }
      this.$root.stopLoading();
    },
    async accept(node) {
      this.hide();
      this.$root.startLoading();
      try {
        await this.$root.papi.post('gridmembers/' + node.id + "/add");
        await this.loadData();
        this.$root.showInfo(this.i18n.gridMemberAcceptSuccess);
      } catch (error) {
        this.$root.showError(error);
      }
      this.$root.stopLoading();
    },
    colorNodeStatus(status) {
      var color = "gray";
      switch (status) {
        case GridMemberRejected: color = "error"; break;
        case GridMemberAccepted: color = "success"; break;
        case GridMemberDenied: color = "warning"; break;
      }
      return color;
    }
  },
}});
