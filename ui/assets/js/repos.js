/** @jsx React.DOM */

Model = {
  didLoadRepos: new Signal(),

  Load: function() {
    var _this = this;

    $.ajax({
      url: '/api/v1/repos',
      dataType: 'json',
      success: function(data) {
        _this.repos = data;
        _this.didLoadRepos.raise(_this, _this.repos);
      },
      error: function(xhr, status, err) {
        // TODO(knorton): Fix these
        console.error(err);
      }
    });
  }
};

var App = React.createClass({
  componentWillMount: function() {
    var _this = this;
    Model.didLoadRepos.tap(function(model, repos) {
      _this.setState({ repos: repos });
    });
  },

  render: function() {

    return (
      <div>
        {this.state.repos.map(function(repo) {
          return (<p>repo</p>);
        })}
      </div>
    );
  }
});
